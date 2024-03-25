package main

import (
	fptr10 "atolserver/fptr"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
)

const Version_of_program = "2024_03_25_01"

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
	Positions []TPositionTask `json:"positions"`
	Cash      float64         `json:"cash"`
	Beznal    float64         `json:"beznal"`
	Return    bool            `json:"return"`
	Cassir    string          `json:"cassir"`
}

type TOperator struct {
	Name  string `json:"name"`
	Vatin string `json:"vatin,omitempty"`
}

type TTaxNDS struct {
	Type string `json:"type,omitempty"`
}

type TPositionTask struct {
	Type  string  `json:"type"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
	//Quantity        float64  `json:"quantity"`
	Quantity        string   `json:"quantity"`
	Amount          float64  `json:"amount"`
	MeasurementUnit string   `json:"measurementUnit"`
	PaymentMethod   string   `json:"paymentMethod"`
	PaymentObject   string   `json:"paymentObject"`
	Tax             *TTaxNDS `json:"tax,omitempty"`
}

type TPosition struct {
	Type     string  `json:"type"`
	Name     string  `json:"name"`
	Price    float64 `json:"price"`
	Quantity float64 `json:"quantity"`
	//Quantity        string   `json:"quantity"`
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

var glKassirName = flag.String("kassir", "админ", "имя, фамилия кассира")

// var fptr *fptr10.IFptr
func main() {
	fmt.Printf("Запуск сервера печати чеков на порту 8080. Версия программы: %v\n", Version_of_program)
	flag.Parse()

	fptrfocloseshift, _ := fptr10.NewSafe()
	connected, typeconn := connectWithKassa(fptrfocloseshift, 0, "", "")
	if !connected {
		panic("ошибка подключения к ККТ")
	}
	cassir := *glKassirName
	if cassir == "" {
		cassir = "админ"
	}
	fmt.Println("Успешное подключение к ККТ", typeconn)
	_, err := checkCloseShift(fptrfocloseshift, true, cassir, true)
	if err != nil {
		errorDescr := fmt.Sprintf("ошибка (%v) закрытия смены", err)
		panic(errorDescr)
	}
	fptrfocloseshift.Close()
	fptrfocloseshift.Destroy()

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
	var summOfCheck float64
	for _, pos := range task.Positions {
		Item := new(TPosition)
		Item.Type = "position"
		Item.Name = pos.Name
		quant, err := strconv.ParseFloat(pos.Quantity, 64)
		if err != nil {
			errDescr := fmt.Sprintf("ошибка (%v) парсинга (%v) поля количества", err, pos.Quantity)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(errDescr))
			fmt.Println(errDescr)
			return
		}
		Item.Quantity = quant
		Item.Price = pos.Price
		Amount := Item.Price * quant
		summOfCheck = summOfCheck + Amount
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
	if len(check.Payments) == 0 {
		check.Payments = append(check.Payments, TPayment{"cash", summOfCheck})
	}
	jsonCheck, err := json.Marshal(check)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintln(err)))
		fmt.Println(fmt.Sprintln(err))
		return
	}
	jsonAnswer, err := sendComandeAndGetAnswerFromKKT(string(jsonCheck), check.Operator.Name)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintln(err)))
		fmt.Println(fmt.Sprintln(err))
		return
	}
	fmt.Fprint(w, jsonAnswer)
}

// func sendComandeAndGetAnswerFromKKT(fptr *fptr10.IFptr, comJson string) (string, error) {
func sendComandeAndGetAnswerFromKKT(comJson, cassir string) (string, error) {
	//return "", nil
	//qqq := "{\"type\": \"reportX\", \"operator\": {\"name\": \"Иванов\"}"
	fmt.Println("comJson", comJson)
	fptr, _ := fptr10.NewSafe()
	defer fptr.Destroy()
	connected, typeconn := connectWithKassa(fptr, 0, "", "")
	if !connected {
		return "", fmt.Errorf("ошибка подключения к ККТ")
	}
	if cassir == "" {
		cassir = "админ"
	}
	fmt.Println("Успешное подключение к ККТ", typeconn)
	_, err := checkCloseShift(fptr, true, cassir, false)
	if err != nil {
		errorDescr := fmt.Sprintf("ошибка (%v). Смена истекла", err)
		//logsmap[LOGERROR].Println(errorDescr)
		return "", errors.New(errorDescr)
	}
	shiftOpenned, err := checkOpenShift(fptr, true, cassir)
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
	fmt.Println("main_command=", comJson)
	fptr.SetParam(fptr10.LIBFPTR_PARAM_JSON_DATA, comJson)
	fptr.ProcessJson()
	result := fptr.GetParamString(fptr10.LIBFPTR_PARAM_JSON_DATA)
	fmt.Println("res_main=", result)
	disconnectWithKKT(fptr, true)
	fmt.Println("result", result)
	return result, nil
}

func sendComandeAndGetAnswerFromKKT__FPTR(fptr *fptr10.IFptr, comJson string) (string, error) {
	fmt.Println("command=", comJson)
	fptr.SetParam(fptr10.LIBFPTR_PARAM_JSON_DATA, comJson)
	fptr.ProcessJson()
	result := fptr.GetParamString(fptr10.LIBFPTR_PARAM_JSON_DATA)
	fmt.Println("result=", result)
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

func checkCloseShift(fptr *fptr10.IFptr, closeShiftIfClose bool, kassir string, closeifopened bool) (bool, error) {
	//logginInFile("получаем статус ККТ")
	getStatusKKTJson := "{\"type\": \"getShiftStatus\"}"
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
	//проверяем - не истёк ли таймаут смена
	var answerOfGetStatusofShift TAnswerGetStatusOfShift
	err = json.Unmarshal([]byte(resgetStatusKKT), &answerOfGetStatusofShift)
	if err != nil {
		errorDescr := fmt.Sprintf("ошибка (%v) распарсивания (%v) статуса кассы", err, resgetStatusKKT)
		fmt.Println(errorDescr)
		//logsmap[LOGERROR].Println(errorDescr)
		return false, err
	}
	if answerOfGetStatusofShift.ShiftStatus.State == "closed" {
		return true, nil
	}
	if (answerOfGetStatusofShift.ShiftStatus.State == "opened") && (closeifopened) {
		if kassir == "" {
			kassir = "админ"
		}
		jsonCloseShift := fmt.Sprintf("{\"type\": \"closeShift\", \"operator\": {\"name\": \"%v\"}}", kassir)
		resCloseShift, err := sendComandeAndGetAnswerFromKKT__FPTR(fptr, jsonCloseShift)
		if err != nil {
			errorDescr := fmt.Sprintf("ошибка (%v) - не удалось закрыть смену", err)
			fmt.Println(errorDescr)
			return false, errors.New(errorDescr)
		}
		if !successCommand(resCloseShift) {
			errorDescr := fmt.Sprintf("ошибка (%v) - не удалось закрыть смену", resCloseShift)
			fmt.Println(errorDescr)
			return false, errors.New(errorDescr)
		}
	}
	if answerOfGetStatusofShift.ShiftStatus.State == "expired" {
		if closeShiftIfClose {
			if kassir == "" {
				errorDescr := "не указано имя кассира для закрытия смены"
				fmt.Println(errorDescr)
				return false, errors.New(errorDescr)
			}
			jsonCloseShift := fmt.Sprintf("{\"type\": \"closeShift\",\"operator\": {\"name\": \"%v\"}}", kassir)
			resCloseShift, err := sendComandeAndGetAnswerFromKKT__FPTR(fptr, jsonCloseShift)
			if err != nil {
				errorDescr := fmt.Sprintf("ошбика (%v) - не удалось закрыть смену", err)
				fmt.Println(errorDescr)
				return false, errors.New(errorDescr)
			}
			if !successCommand(resCloseShift) {
				errorDescr := fmt.Sprintf("ошбика (%v) - не удалось закрыть смену", resCloseShift)
				fmt.Println(errorDescr)
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
	getStatusKKTJson := "{\"type\": \"getShiftStatus\"}"
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
		errorDescr := fmt.Sprintf("ошибка (%v) распарсивания (%v) статуса кассы", err, resgetStatusKKT)
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
