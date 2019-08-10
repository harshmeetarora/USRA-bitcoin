package main
import (
//  "bufio"
  "fmt"
  "strings"
  "bytes"
  "log"
  "net/http"
  //"time"
  "strconv"
  "io/ioutil"
  "os"
//  "encoding/csv"
  "encoding/json"
)
var rpcnode = "http://127.0.0.1:8332"
var client = http.Client {}
type Call struct {
  Jsonrpc string        `json:"jsonrpc"`
  Id      string        `json:"id"`
  Method  string        `json:"method"`
  Params  []interface{} `json:"params"`
}
//type currItem struct {
//  account int
//  txid      string 
//  vin_txid  []string  
//  utxos  []string 
//}

type txData struct {
	inputAcc int
	outputAcc int
	txid string
	value uint64
}
func dorpc (method string, params []interface{}) (map[string]interface{}) {
  requestJson, _ := json.Marshal(&Call{Jsonrpc:"1.0", Id : "0", Method: method, Params: params})
  hreq, _ := http.NewRequest("POST", rpcnode, bytes.NewBuffer(requestJson))
  hreq.SetBasicAuth("__cookie__", "aaf85c8cab0dbf81d2cbd4fa5fcab61b8879c75e3de323430803d9f5d9c3b55f")
  resp, _ := client.Do(hreq)
  responseJson, _ := ioutil.ReadAll(resp.Body)
  var respMap map[string]interface{}
  _ = json.Unmarshal(responseJson, &respMap)
  return respMap
}
func rpcgetblock (blockhash string) (map[string]interface{}) {
  response := dorpc("getblock", []interface{}{blockhash, 2})
  result, _ := response["result"].(map[string]interface{})
  return result
}
func rpcgetblockhash (block int) (string) {
  response := dorpc("getblockhash", []interface{}{block})
  result, _ := response["result"].(string)
  return result
}
func rpcgetUTXOs(blockheight int, utxoAccDictptr *map[string] *int, txidOutAccptr *map[string] *int, txidInAccptr *map[string] **int, txidValueptr*map[string] uint64){
	if (blockheight%2000 == 0) {
			log.Println(blockheight)
	}
	block := rpcgetblock(rpcgetblockhash(blockheight))
	tx, ok := (block["tx"]).([]interface{})
	if !ok { log.Println("error")}
//	var output_add []string
	dictSize := len(*utxoAccDictptr)+1
	for txNum, transaction := range tx {
			transactionT, _ := transaction.(map[string]interface{})
			txidT, _ := transactionT["txid"].(string)
			txid := fmt.Sprintf("%v", txidT)
			currOutAcc := (dictSize + txNum)
			currOutAccptr := &currOutAcc
			output_add := getOutputs(txid, transactionT, txidValueptr)
             // log.Println(output_add)
			vinTxids := getIntputsTxids(transactionT)
			updateOuts(txid, output_add, utxoAccDictptr,txidOutAccptr, currOutAccptr)
			updateIns(txid, vinTxids, txidOutAccptr, txidInAccptr)
	}
}
func updateOuts(txid string, txOutputs []string, utxoAccDictptr *map[string] *int, txidOutAccptr *map[string] *int, currOutAccptr *int){
	for _, utxo := range txOutputs {
		if val, ok := (*utxoAccDictptr)[utxo]; ok {
			*val = *currOutAccptr
	//		log.Println("merge out : ", txid)
		//	log.Println(*val)
		//	log.Println(utxo)
		}else{
			(*utxoAccDictptr)[utxo]= currOutAccptr
		}
	}
	(*txidOutAccptr)[txid] = currOutAccptr
//	log.Println(*utxoAccDictptr)
}
func updateIns (txid string, vinTxids []string, txidOutDictptr *map [string] *int, txidInDictptr *map[string] **int){
	currInAcc := 0
	var currInAccptr **int
	currInAccP1 := &currInAcc
	currInAccptr = &currInAccP1
	for _, vin_txid := range vinTxids {
		if val, ok := (*txidOutDictptr)[vin_txid]; ok{
			*currInAccptr = val
	//		log.Println("merge in : ", txid)
		}
	}
	(*txidInDictptr)[txid] = currInAccptr
}

func getOutputs (txid string, transaction map[string]interface{}, txidValueptr *map[string] uint64) ([]string) {
  value := 0.0
  vout, _ := transaction["vout"].([]interface{})
  var output_add []string
  for _, output := range vout {
    outputT, _ := output.(map[string]interface{})
    value += outputT["value"].(float64)
    sPKT, _ := outputT["scriptPubKey"].(map[string]interface{})
    output_addresses, ok := sPKT["addresses"]
    if !ok{log.Println("Error")}
    in := formatString(output_addresses)
    output_add = append(output_add, in)
  }
// log.Println(value)
  (*txidValueptr)[txid] = valueFormatter(value)
  return output_add
}

func getIntputsTxids (transaction map[string]interface{}) ([]string) {
  vin, _ := transaction["vin"].([]interface{})
  var input_txid []string
  for _, input := range vin {
    inputT, _ := input.(map[string]interface{})
    var inputTxid string
    inputTxid, ok := inputT["txid"].(string)
    if !ok {
      inputTxid = "coinbase"
    }
    input_txid = append(input_txid, inputTxid)
  }
  return input_txid
}

func formatString(address interface{})(string){
	inputString := fmt.Sprintf("%v", address)
	trimSuff := strings.TrimSuffix(inputString, "]")
	trimPre := strings.TrimPrefix(trimSuff, "[")
	return trimPre
}

//func writeMatrixMarketFile(data []string) {
//    file, err := os.Create("result.txt")
//    checkError("Cannot create file", err)
//    defer file.Close()
//
//    writer := bufio.NewWriter(file)
//    defer writer.Flush()

//    for _, value := range data {
  //      err := writer.Write(value)
    //    checkError("Cannot write to file", err)
   // }
//}



func makeMMString(inAcc int, outAcc int, val uint64)(string){
	mmStr := strconv.Itoa(inAcc) + " " + strconv.Itoa(outAcc)+ " " + strconv.FormatUint(val, 10)
	return mmStr
}
func valueFormatter (floatVal float64)(uint64){
	s := fmt.Sprintf("%.8f", floatVal)
	f, _ := strconv.ParseFloat(s, 64)
	var i uint64 = uint64(f*100000000.0)
	return i
}
//func strToSatoshi (strVal string)(uint64){}


func main(){

	utxoAccDict := map[string] *int{}
	txidOutDict := map[string] *int{}
	txidInDict := map[string] **int{}
	txidValue := map[string] uint64{}
//      var data []interface{}
	for i := 0; i<123456; i++{
		rpcgetUTXOs(i, &utxoAccDict, &txidOutDict, &txidInDict, &txidValue)
//		log.Println(utxoAccDict)
	}
	f, err := os.Create("output")
	if err != nil {
		fmt.Println(err)
                f.Close()
        return
	}
        metaF, err := os.Create("metadata")
        if err != nil {
                fmt.Println(err)
                metaF.Close()
        return
        }


	//var data []txData
	//var strData []string
//	dataSize := len (txidOutDict)
//	headerStr := makeMMString(dataSize, dataSize, uint64(dataSize*dataSize))
//	fmt.Fprintln(f, headerStr)
	for txid, outptr := range txidOutDict{
//		item := txData {**(txidInDict[txid]), *outptr, txid, txidValue[txid]}
		strItem := makeMMString(**(txidInDict[txid]), *outptr, txidValue[txid])
		//strData = append (strData, strItem)
		metaStr := strItem + " " + txid
		fmt.Fprintln(f, strItem)
		if err != nil {
			fmt.Println(err)
			return
		}
                fmt.Fprintln(metaF, metaStr)
                if err != nil {
                        fmt.Println(err)
                        return
                }

	//	data = append (data, item)
	}
	f.Close()
	metaF.Close()
	//log.Println(txidInDict)
	//log.Println(data)
	//writeMatrixMarketFile(strData)
}
