package main
import (
  "fmt"
  "strings"
  "bytes"
  "log"
  "net/http"
  //"time"
//  "strconv"
  "io/ioutil"
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
	value float64
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
func rpcgetUTXOs(blockheight int, utxoAccDictptr *map[string] *int, txidOutAccptr *map[string] *int, txidInAccptr *map[string] **int, txidValueptr*map[string] float64){
	if (blockheight%2000 == 0) {
			log.Println(blockheight)
	}
	block := rpcgetblock(rpcgetblockhash(blockheight))
	tx, ok := (block["tx"]).([]interface{})
	if !ok { log.Println("error")}
//	var output_add []string
	dictSize := len(*utxoAccDictptr)
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
	currInAcc := -1
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

func getOutputs (txid string, transaction map[string]interface{}, txidValueptr *map[string] float64) ([]string) {
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
  (*txidValueptr)[txid] = value
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

func main(){

	utxoAccDict := map[string] *int{}
	txidOutDict := map[string] *int{}
	txidInDict := map[string] **int{}
	txidValue := map[string] float64{}
//      var data []interface{}
	for i := 0; i<12; i++{
		rpcgetUTXOs(i, &utxoAccDict, &txidOutDict, &txidInDict, &txidValue)
//		log.Println(utxoAccDict)
	}
	var data []txData
	for txid, outptr := range txidOutDict{
		item := txData {**(txidInDict[txid]), *outptr, txid, txidValue[txid]}
		data = append (data, item)
	}
	//log.Println(txidInDict)
	log.Println(data)
}
