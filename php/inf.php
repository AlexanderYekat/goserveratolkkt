<?php
// Определение констант
define('LIBFPTR_PARAM_COMMODITY_NAME', 65631);
define('LIBFPTR_PARAM_PRICE', 65632);
define('LIBFPTR_PARAM_QUANTITY', 65633);
define('LIBFPTR_PARAM_TAX_TYPE', 65569);
define('LIBFPTR_PARAM_PAYMENT_TYPE', 65564);
define('LIBFPTR_PARAM_PAYMENT_SUM', 65565);
define('LIBFPTR_TAX_NO', 6);
define('LIBFPTR_PT_CASH', 0);
define('LIBFPTR_PT_ELECTRONICALLY', 1);
define('LIBFPTR_PARAM_DOCUMENT_CLOSED', 65644); 
define('LIBFPTR_PARAM_MEASUREMENT_UNIT', 65851);
define('LIBFPTR_PARAM_REPORT_TYPE', 65546);
define('LIBFPTR_RT_CLOSE_SHIFT', 2);
define('LIBFPTR_PARAM_RECEIPT_TYPE', 65545);  // Признак расчета
define('LIBFPTR_RT_SELL', 1);                 // Приход
define('LIBFPTR_RT_SELL_RETURN', 2);          // Возврат прихода
define('LIBFPTR_SETTING_PORT', "Port");         // Порт подключения

$zakid = $_GET["zakid"];
//$zakid = "162279185"; //номер заказа
$zapr = json_decode(file_get_contents("https://abcp56606.public.api.abcp.ru/cp/order?userlogin=api@abcp56606&userpsw=a7b7b4ec215810e3900225c5de3e917f&number=".$zakid));

class TCheck
{
    public $cash;
    public $beznal;
    public $return;
    public $cassir;
    public $positions;
}

$zapr->cassir = "Исаев Игорь";
$chcek = new TCheck();
$chcek->cash = $zapr->cash; //сумма наличнымим - название полей?????????
$chcek->beznal = $zapr->beznal; //сумма безналом - название полей?????????
$chcek->cassir=$zapr->cassir; //имя кассира - название полей?????????
$chcek->return=$zapr->return; //является ли чек - чеком возврата - название полей?????????
$chcek->positions = array (
);

	foreach ($zapr->positions as $item) {
    $newitem["name"] = chop($item->description) . " " . chop($item->brand); //наименование товара
    $newitem["quantity"] = $item->quantityFinal; 
	//echo $newitem["quantity"];
    $newitem["price"] = $item->priceOut;
    array_push($chcek->positions, $newitem);
}

function printCheck($fptr, $chcek) {
    try {
        // Инициализация драйвера
        $fptr = new COM("AddIn.Fptr10") or die("Не удалось инициализировать объект Excel");
        $version = $fptr->version;
        echo $version;
    
        // Открытие смены
        //openShift($fptr, $chcek->cassir);
    
        // Настройка подключения к ККТ
        $fptr->SetSingleSetting(LIBFPTR_SETTING_PORT, "1"); //USB
        $fptr->applySingleSettings();
        
        // Подключение к ККТ
        $fptr->open();
        if (!$fptr->isOpened()) {
            throw new Exception("Не удалось подключиться к ККТ");
        }
    
        // Открытие чека
        $fptr->setParam(1021, $chcek->cassir); // Кассир
        $fptr->operatorLogin();
        
        if ($chcek->return) {
            $fptr->setParam(LIBFPTR_PARAM_RECEIPT_TYPE, LIBFPTR_RT_SELL_RETURN); // Признак расчета (возврат)
            $fptr->openReceipt();
        } else {
            $fptr->setParam(LIBFPTR_PARAM_RECEIPT_TYPE, LIBFPTR_RT_SELL); // Признак расчета (приход)
            $fptr->openReceipt();
        }
    
        // Добавление позиций
        foreach ($chcek->positions as $position) {
            $fptr->setParam(LIBFPTR_PARAM_COMMODITY_NAME, $position["name"]); // Наименование 
            $fptr->setParam(LIBFPTR_PARAM_PRICE, $position["price"]); // Цена
            $fptr->setParam(LIBFPTR_PARAM_QUANTITY, $position["quantity"]); // Количество
            $fptr->setParam(1212, 1); // товар
            $fptr->setParam(1214, 4); // полный рачсет
            $fptr->setParam(LIBFPTR_PARAM_MEASUREMENT_UNIT, 0); // штуки
            $fptr->setParam(LIBFPTR_PARAM_TAX_TYPE, LIBFPTR_TAX_NO); // НДС не облагается
            $fptr->registration();
        }
    
        // Оплата
        if ($chcek->cash > 0) {
            $fptr->setParam(LIBFPTR_PARAM_PAYMENT_TYPE, LIBFPTR_PT_CASH); // Наличная оплата
            $fptr->setParam(LIBFPTR_PARAM_PAYMENT_SUM, $chcek->cash);
            $fptr->payment();
        }
        if ($chcek->beznal > 0) {
            $fptr->setParam(LIBFPTR_PARAM_PAYMENT_TYPE, LIBFPTR_PT_ELECTRONICALLY); // Безналичная оплата
            $fptr->setParam(LIBFPTR_PARAM_PAYMENT_SUM, $chcek->beznal);        
            $fptr->payment();
        }
        if (count($chcek->positions) == 0) {
            $fptr->setParam(LIBFPTR_PARAM_PAYMENT_TYPE, LIBFPTR_PT_CASH);
            $fptr->setParam(LIBFPTR_PARAM_PAYMENT_SUM, $chcek->cash);
            $fptr->payment();
        }
    
        // Закрытие чека
        if (!$fptr->closeReceipt()) {
            print($fptr->errorDescription());
            throw new Exception("Ошибка при закрытии чека: " . $fptr->errorDescription());
        }
    
        while ($fptr->checkDocumentClosed()<0) {
            print($fptr->errorDescription());
            //echo $fptr->errorDescription();
            continue;
        }
        
        if (!$fptr->getParamBool(LIBFPTR_PARAM_DOCUMENT_CLOSED)) {
            // Документ не закрылся. Требуется его отменить (если это чек) и сформировать заново
            $fptr->cancelReceipt();
            throw new Exception("Ошибка при закрытии чека: " . $fptr->errorDescription());
        }
    
        // Отключение от ККТ
        $fptr->close();
        
        print("Чек успешно напечатан");
    
        // Закрытие смены
        //closeShift($fptr, $chcek->cassir);
    
    } catch (Exception $e) {
        print($fptr->errorDescription());
        echo "Ошибка при печати чека: " . $e->getMessage();
        if (isset($fptr)) {
            $fptr->close();
        }
    }
}    

function openShift($fptr, $cassir) {
    $fptr->setParam(1021, $cassir); // Установка кассира
    $fptr->operatorLogin();
    if (!$fptr->openShift()) {
        throw new Exception("Ошибка при открытии смены: " . $fptr->errorDescription());
    }
    echo "Смена успешно открыта";
}

printCheck($fptr, $chcek);

function closeShift($fptr, $cassir) {
    $fptr->setParam(1021, $cassir); // Установка кассира
    $fptr->operatorLogin();
    $fptr->setParam(LIBFPTR_PARAM_REPORT_TYPE, LIBFPTR_RT_CLOSE_SHIFT);
    if (!$fptr->report()) {
        throw new Exception("Ошибка при закрытии смены: " . $fptr->errorDescription());
    }
    while ($fptr->checkDocumentClosed()!=0) {
        print($fptr->errorDescription());
        continue;
    }

    if (!$fptr->getParamBool(LIBFPTR_PARAM_DOCUMENT_CLOSED)) {
        $fptr->cancelReceipt();
        throw new Exception("Ошибка при закрытии смены: " . $fptr->errorDescription());
    }
    
    echo "Смена успешно закрыта";
}

header("Location: http://ide.rs-truck.ru/kassa_td.php");
?>