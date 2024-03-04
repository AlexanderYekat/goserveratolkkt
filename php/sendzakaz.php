<html>
<head>
    <title>Test</title>
</head>
<body>
<?php

class TZak 
{
    public $cash;
    public $beznal;
    public $return;
    public $cassir;
    //public function __construct()
    public $positions;
}

class TCheck
{
    public $cash;
    public $beznal;
    public $return;
    public $cassir;
    //public function __construct()
    public $positions;
}


class item {
    public $description;
    public $brand;
    public $quntityFinal;
    public $priceOut;
};


//$fptr = new com("AddIn.Fptr10");
//print(fptr.version());

$it1 = new item();
$it1->description = "наименование товара 1";
$it1->brand = "brand1";
$it1->quntityFinal = "3";
$it1->priceOut = 200;

$it2 = new item();
$it2->description = "наименование товара 2";
$it2->brand = "brand2";
$it2->quntityFinal = "1";
$it2->priceOut = 250;

$zapr = new TZak();
$zapr->cash = 0;
$zapr->beznal = 0;
$zapr->cassir  = "Иванов";
$zapr->return = False;

$zapr->positions = array (
    $it1,
    $it2
);

$chcek = new TCheck();
$chcek->cash = $zapr->cash; //сумма наличнымим - название полей?????????
$chcek->beznal = $zapr->beznal; //сумма безналом - название полей?????????
$chcek->cassir=$zapr->cassir; //имя кассира - название полей?????????
$chcek->return=$zapr->return; //является ли чек - чеком возврата - название полей?????????
$chcek->positions = array (
);


foreach ($zapr->positions as $item) {
    $newitem["name"] = chop($item->description) . " " . chop($item->brand); //наименование товара
    $newitem["quantity"] = $item->quntityFinal;
    $newitem["price"] = $item->priceOut;
    array_push($chcek->positions, $newitem);

}
$data_string = json_encode($chcek);
print($data_string ); 
$url = "localhost:8080";
$curl = curl_init($url);
curl_setopt($curl, CURLOPT_HEADER, false);
curl_setopt($curl, CURLOPT_RETURNTRANSFER, true);
curl_setopt($curl, CURLOPT_HTTPHEADER,
        array("Content-type: application/json"));
curl_setopt($curl, CURLOPT_POST, true);
curl_setopt($curl, CURLOPT_POSTFIELDS, $data_string);
$json_response = curl_exec($curl);
$response = json_decode($json_response, true);
//print($response);
//print($json_response);
$status = curl_getinfo($curl, CURLINFO_HTTP_CODE);
curl_close($curl);
if ( $status != 200 ) {
    print("чек не расчпечатан по причине:");
    if ($status == 0) {
        print("нет связи с программой печти чеков");
    }
    if ($status == 500) {
        //$response = json_decode($json_response, true);
        print($json_response);        
    }
} else {
    print($json_response);
    //$response = json_decode($json_response, true);
    print("Чек распечатан");
}
?>
</body>
</html>