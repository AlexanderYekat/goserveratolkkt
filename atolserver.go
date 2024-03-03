package main

import (
	fptr10 "atolserver/fptr"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
)

type TRepotyAtolKKT struct {
	Type     string    `json:"type"`
	Operator TOperator `json:"operator"`
}

type TAnswerGetStatusOfShift struct {
	ShiftStatus TShiftStatus `json:"shiftStatus"`
}
type TShiftStatus struct {
	DocumentsCount int    `json:"documentsCount"`
	ExpiredTime    string `json:"expiredTime"`
	Number         int    `json:"number"`
	State          string `json:"state"`
}

type TTask struct {
	Positions []TPosition `json:"positions"`
	Cash      float64     `json:"cash"`
	Beznal    float64     `json:"beznal"`
	Return    bool        `json:"return"`
	Cassir    string      `json:"cassir"`
}

type TOperator struct {
	Name  string `json:"name"`
	Vatin string `json:"vatin,omitempty"`
}

type TTaxNDS struct {
	Type string `json:"type,omitempty"`
}

type TPosition struct {
	Type            string   `json:"type"`
	Name            string   `json:"name"`
	Price           float64  `json:"price"`
	Quantity        float64  `json:"quantity"`
	Amount          float64  `json:"amount"`
	MeasurementUnit string   `json:"measurementUnit"`
	PaymentMethod   string   `json:"paymentMethod"`
	PaymentObject   string   `json:"paymentObject"`
	Tax             *TTaxNDS `json:"tax,omitempty"`
}

type TPayment struct {
	Type string  `json:"type"`
	Sum  float64 `json:"sum"`
}

type TCheck struct {
	Type           string `json:"type"` //sellCorrection - чек коррекции прихода
	Electronically bool   `json:"electronically"`
	TaxationType   string `json:"taxationType,omitempty"`
	//ClientInfo           TClientInfo `json:"clientInfo"`
	Operator TOperator `json:"operator"`
	//Items                []TPosition `json:"items"`
	Items    []TPosition `json:"items"` //либо TTag1192_91, либо TPosition
	Payments []TPayment  `json:"payments"`
	Total    float64     `json:"total,omitempty"`
}

// var fptr *fptr10.IFptr
func main() {
	fmt.Println("Запуск сервера на порту 8080")
	http.HandleFunc("/", handler)
	http.ListenAndServe(":8080", nil)
}
func handler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}
	//jsontask := string(body)
	var task TTask
	err = json.Unmarshal(body, &task)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintln(err)))
		fmt.Println(fmt.Sprintln(err))
		return
	}
	log.Println(string(body))
	var check TCheck
	check.Type = "sell"
	if task.Return {
		check.Type = "sellReturn"
	}
	check.Operator.Name = task.Cassir
	if task.Cassir == "" {
		check.Operator.Name = "Иванов"
	}
	for _, pos := range task.Positions {
		Item := new(TPosition)
		Item.Name = pos.Name
		Item.Quantity = pos.Quantity
		Item.Price = pos.Price
		Amount := Item.Price * Item.Quantity
		Item.Amount = Amount
		Item.MeasurementUnit = "piece"
		Item.PaymentMethod = "fullPayment"
		Item.PaymentObject = "commodity"
		Item.Tax = new(TTaxNDS)
		Item.Tax.Type = "none"
		check.Items = append(check.Items, *Item)
	}
	if task.Cash > 0 {
		check.Payments = append(check.Payments, TPayment{"cash", task.Cash})
	}
	if task.Beznal > 0 {
		check.Payments = append(check.Payments, TPayment{"prepayment", task.Beznal})
	}
	jsonCheck, err := json.Marshal(check)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintln(err)))
		fmt.Println(fmt.Sprintln(err))
		return
	}
	jsonAnswer, err := sendComandeAndGetAnswerFromKKT(string(jsonCheck))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintln(err)))
		fmt.Println(fmt.Sprintln(err))
		return
	}
	fmt.Fprintf(w, jsonAnswer)
}

// func sendComandeAndGetAnswerFromKKT(fptr *fptr10.IFptr, comJson string) (string, error) {
func sendComandeAndGetAnswerFromKKT(comJson string) (string, error) {
	//return "", nil
	//qqq := "{\"type\": \"reportX\", \"operator\": {\"name\": \"Иванов\"}"
	fmt.Println("comJson", comJson)
	fptr, _ := fptr10.NewSafe()
	defer fptr.Destroy()
	connected, typeconn := connectWithKassa(fptr, 0, "", "")
	if !connected {
		return "", fmt.Errorf("Ошибка подключения к ККТ")
	}
	fmt.Println("Успешное подключение к ККТ", typeconn)
	shiftOpenned, err := checkOpenShift(fptr, true, "админ")
	if err != nil {
		errorDescr := fmt.Sprintf("ошибка (%v). Смена не открыта", err)
		//logsmap[LOGERROR].Println(errorDescr)
		return "", errors.New(errorDescr)
	}
	if !shiftOpenned {
		errorDescr := fmt.Sprintf("ошибка (%v) - смена не открыта", err)
		//logsmap[LOGERROR].Println(errorDescr)
		return "", errors.New(errorDescr)
	}

	fptr.SetParam(fptr10.LIBFPTR_PARAM_JSON_DATA, comJson)
	result := fptr.GetParamString(fptr10.LIBFPTR_PARAM_JSON_DATA)
	disconnectWithKKT(fptr, true)
	fmt.Println("result", result)
	return result, nil
}

func sendComandeAndGetAnswerFromKKT__FPTR(fptr *fptr10.IFptr, comJson string) (string, error) {
	fptr.SetParam(fptr10.LIBFPTR_PARAM_JSON_DATA, comJson)
	result := fptr.GetParamString(fptr10.LIBFPTR_PARAM_JSON_DATA)
	return result, nil
}

func connectWithKassa(fptr *fptr10.IFptr, comportint int, ipaddresskktper, ipaddresssrvkktper string) (bool, string) {
	typeConnect := ""
	fptr.SetSingleSetting(fptr10.LIBFPTR_SETTING_MODEL, strconv.Itoa(fptr10.LIBFPTR_MODEL_ATOL_AUTO))
	if ipaddresssrvkktper != "" {
		fptr.SetSingleSetting(fptr10.LIBFPTR_SETTING_REMOTE_SERVER_ADDR, ipaddresssrvkktper)
		typeConnect = fmt.Sprintf("через сервер ККТ по IP %v", ipaddresssrvkktper)
	}
	if comportint == 0 {
		if ipaddresskktper != "" {
			fptr.SetSingleSetting(fptr10.LIBFPTR_SETTING_IPADDRESS, ipaddresskktper)
			typeConnect = fmt.Sprintf("%v по IP %v ККТ", typeConnect, ipaddresskktper)
		} else {
			fptr.SetSingleSetting(fptr10.LIBFPTR_SETTING_PORT, strconv.Itoa(fptr10.LIBFPTR_PORT_USB))
			typeConnect = fmt.Sprintf("%v по USB", typeConnect)
		}
	} else {
		sComPorta := "COM" + strconv.Itoa(comportint)
		typeConnect = fmt.Sprintf("%v по COM порту %v", typeConnect, sComPorta)
		fptr.SetSingleSetting(fptr10.LIBFPTR_SETTING_PORT, strconv.Itoa(fptr10.LIBFPTR_PORT_COM))
		fptr.SetSingleSetting(fptr10.LIBFPTR_SETTING_COM_FILE, sComPorta)
		fptr.SetSingleSetting(fptr10.LIBFPTR_SETTING_BAUDRATE, strconv.Itoa(fptr10.LIBFPTR_PORT_BR_115200))
	}
	fptr.ApplySingleSettings()
	fptr.Open()
	return fptr.IsOpened(), typeConnect
}

func disconnectWithKKT(fptr *fptr10.IFptr, destroyComObject bool) {
	fptr.Close()
	if destroyComObject {
		fptr.Destroy()
	}
}

func checkCloseShift(fptr *fptr10.IFptr, closeShiftIfClose bool, kassir string) (bool, error) {
	//logginInFile("получаем статус ККТ")
	getStatusKKTJson := "{\"type\": \"getDeviceStatus\"}"
	resgetStatusKKT, err := sendComandeAndGetAnswerFromKKT__FPTR(fptr, getStatusKKTJson)
	if err != nil {
		errorDescr := fmt.Sprintf("ошибка (%v) получения статуса кассы", err)
		fmt.Println(errorDescr)
		//logsmap[LOGERROR].Println(errorDescr)
		return false, err
	}
	if !successCommand(resgetStatusKKT) {
		errorDescr := fmt.Sprintf("ошибка (%v) получения статуса кассы", resgetStatusKKT)
		fmt.Println(errorDescr)
		//logsmap[LOGERROR].Println(errorDescr)
		//logginInFile(errorDescr)
		return false, errors.New(errorDescr)
	}
	//logginInFile("получили статус кассы")
	//проверяем - открыта ли смена
	var answerOfGetStatusofShift TAnswerGetStatusOfShift
	err = json.Unmarshal([]byte(resgetStatusKKT), &answerOfGetStatusofShift)
	if err != nil {
		errorDescr := fmt.Sprintf("ошибка (%v) распарсивания статуса кассы", err)
		fmt.Println(errorDescr)
		//logsmap[LOGERROR].Println(errorDescr)
		return false, err
	}
	if answerOfGetStatusofShift.ShiftStatus.State == "closed" {
		return true, nil
	}
	if answerOfGetStatusofShift.ShiftStatus.State == "expired" {
		if closeShiftIfClose {
			if kassir == "" {
				errorDescr := "не указано имя кассира для закрытия смены"
				fmt.Println(errorDescr)
				//logsmap[LOGERROR].Println(errorDescr)
				return false, errors.New(errorDescr)
			}
			jsonCloseShift := fmt.Sprintf("{\"type\": \"closeShift\",\"operator\": {\"name\": \"%v\"}}", kassir)
			resOpenShift, err := sendComandeAndGetAnswerFromKKT__FPTR(fptr, jsonCloseShift)
			if err != nil {
				errorDescr := fmt.Sprintf("ошбика (%v) - не удалось закрыть смену", err)
				fmt.Println(errorDescr)
				//logsmap[LOGERROR].Println(errorDescr)
				return false, errors.New(errorDescr)
			}
			if !successCommand(resOpenShift) {
				errorDescr := fmt.Sprintf("ошбика (%v) - не удалось закрыть смену", resOpenShift)
				fmt.Println(errorDescr)
				//logsmap[LOGERROR].Println(errorDescr)
				return false, errors.New(errorDescr)
			}
		} else {
			return false, nil
		}
	}
	return true, nil
} //checkCloseShift

func checkOpenShift(fptr *fptr10.IFptr, openShiftIfClose bool, kassir string) (bool, error) {
	//logginInFile("получаем статус ККТ")
	getStatusKKTJson := "{\"type\": \"getDeviceStatus\"}"
	resgetStatusKKT, err := sendComandeAndGetAnswerFromKKT__FPTR(fptr, getStatusKKTJson)
	if err != nil {
		errorDescr := fmt.Sprintf("ошибка (%v) получения статуса кассы", err)
		fmt.Println(errorDescr)
		//logsmap[LOGERROR].Println(errorDescr)
		return false, err
	}
	if !successCommand(resgetStatusKKT) {
		errorDescr := fmt.Sprintf("ошибка (%v) получения статуса кассы", resgetStatusKKT)
		fmt.Println(errorDescr)
		//logsmap[LOGERROR].Println(errorDescr)
		//logginInFile(errorDescr)
		return false, errors.New(errorDescr)
	}
	//logginInFile("получили статус кассы")
	//проверяем - открыта ли смена
	var answerOfGetStatusofShift TAnswerGetStatusOfShift
	err = json.Unmarshal([]byte(resgetStatusKKT), &answerOfGetStatusofShift)
	if err != nil {
		errorDescr := fmt.Sprintf("ошибка (%v) распарсивания статуса кассы", err)
		fmt.Println(errorDescr)
		//logsmap[LOGERROR].Println(errorDescr)
		return false, err
	}
	if answerOfGetStatusofShift.ShiftStatus.State == "expired" {
		errorDescr := "ошибка - смена на кассе уже истекла. Закройте смену"
		fmt.Println(errorDescr)
		//logsmap[LOGERROR].Println(errorDescr)
		return false, errors.New(errorDescr)
	}
	if answerOfGetStatusofShift.ShiftStatus.State == "closed" {
		if openShiftIfClose {
			if kassir == "" {
				errorDescr := "не указано имя кассира для открытия смены"
				fmt.Println(errorDescr)
				//logsmap[LOGERROR].Println(errorDescr)
				return false, errors.New(errorDescr)
			}
			jsonOpenShift := fmt.Sprintf("{\"type\": \"openShift\",\"operator\": {\"name\": \"%v\"}}", kassir)
			resOpenShift, err := sendComandeAndGetAnswerFromKKT__FPTR(fptr, jsonOpenShift)
			if err != nil {
				errorDescr := fmt.Sprintf("ошбика (%v) - не удалось открыть смену", err)
				fmt.Println(errorDescr)
				//logsmap[LOGERROR].Println(errorDescr)
				return false, errors.New(errorDescr)
			}
			if !successCommand(resOpenShift) {
				errorDescr := fmt.Sprintf("ошбика (%v) - не удалось открыть смену", resOpenShift)
				fmt.Println(errorDescr)
				//logsmap[LOGERROR].Println(errorDescr)
				return false, errors.New(errorDescr)
			}
		} else {
			return false, nil
		}
	}
	return true, nil
} //checkOpenShift

func successCommand(resulJson string) bool {
	res := true
	indOsh := strings.Contains(resulJson, "ошибка")
	indErr := strings.Contains(resulJson, "error")
	if indErr || indOsh {
		res = false
	}
	return res
} //successCommand
