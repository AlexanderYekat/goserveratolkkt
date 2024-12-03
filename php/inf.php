<?


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
    //ошибка печати чека
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
		
		
	header("Location: http://ide.rs-truck.ru/kassa_td.php");		
	
?>