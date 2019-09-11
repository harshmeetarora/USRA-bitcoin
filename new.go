package main
import (
  "fmt"
  "strings"
  "bytes"
  "log"
  "net/http"
  "strconv"
  "io/ioutil"
  "os"
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
func dorpc (method string, params []interface{}) (map[string]interface{}) {
	requestJson, _ := json.Marshal(&Call{Jsonrpc:"1.0", Id : "0", Method: method, Params: params})
	hreq, _ := http.NewRequest("POST", rpcnode, bytes.NewBuffer(requestJson))
	hreq.SetBasicAuth("__cookie__", "1882fdb54e29faa606b7801f13261ce0c7b74c1210baf1312db832a641afaa6c")
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
 
/*
*RPC call, gets block, iterates through each transactions,
*updates input and output account pointers
*gets values for each transaction
*@params: 
*       blockheight
* utxoAccDictptr : pointer to mapping of utxo to output acc
* txidOutAccptr : pointer to mapping of txid to output acc
* txidInAccptr : pointer to mapping of txid to input acc
* txidValueptr : pointer to mapping of txid to transaction value in satoshi
*/
func rpcgetUTXOs(blockheight int, utxoAccDictptr *map[string] *int, txidOutAccptr *map[string] *int, txidInAccptr *map[string] **int, txidValueptr*map[string] uint64, vinAccDictptr *map[string] *int){
	if (blockheight%2000 == 0) {
		log.Println(blockheight)
	}
	block := rpcgetblock(rpcgetblockhash(blockheight))
	tx, ok := (block["tx"]).([]interface{})
	if !ok { log.Println("tx Error")}
	dataSize := len(*txidOutAccptr)
	for txNum, transaction := range tx {
		transactionT, _ := transaction.(map[string]interface{})
		txidT, _ := transactionT["txid"].(string)
		txid := fmt.Sprintf("%v", txidT)
		currOutAcc := txNum+dataSize+2
		currOutAccptr := &currOutAcc
		output_add := getOutputs(txid, transactionT, txidValueptr)
		vinTxids := getIntputsTxids(transactionT)
		updateOuts(txid, output_add, utxoAccDictptr,txidOutAccptr, currOutAccptr)
		mergeVins(txid, vinTxids, vinAccDictptr, currOutAccptr)
		updateIns(txid, vinTxids, txidOutAccptr, txidInAccptr)
	}
}

/*
*return list of utxos for current txid
* maps value of transaction to current txid
*@params: 
*       txid: current txid
* transaction : current transaction object from rpc
* txidValueptr : pointer to mapping of txid to transaction value in satoshi
*@returns:
* array of utxos for current txid
*/
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
	(*txidValueptr)[txid] = valueFormatter(value)
	return output_add
  }

  /*
*return list of utxos for current txid
* maps value of transaction to current txid
*@params: 
* transaction : current transaction object from rpc
*@returns:
* array of vin_txid for current txid
*/
func getIntputsTxids (transaction map[string]interface{}) ([]string) {
	vin, _ := transaction["vin"].([]interface{})
	var vin_txids []string
	for _, input := range vin {
	  inputT, _ := input.(map[string]interface{})
	  var inputTxid string
	  inputTxid, ok := inputT["txid"].(string)
	  if !ok {
		inputTxid = "coinbase"
	  }
	  vin_txids = append(vin_txids, inputTxid)
	}
	return vin_txids
  }

/*
*updates mapping utxo to out account pointers
*maps txid to output account pointer
*@params: 
* txid: current txid
* txOutputs : array of all utxos for current txid
* utxoAccDictptr : pointer to mapping of utxo to output acc
* txidOutAccptr : pointer to mapping of txid to output acc
*/
func updateOuts(txid string, txOutputs []string, utxoAccDictptr *map[string] *int, txidOutAccptr *map[string] *int, currOutAccptr *int){
	for _, utxo := range txOutputs {
			if val, ok := (*utxoAccDictptr)[utxo]; ok {
					*val = *currOutAccptr
			}else{
					(*utxoAccDictptr)[utxo]= currOutAccptr
			}
	}
	(*txidOutAccptr)[txid] = currOutAccptr
}
/*
*megres input accounts based on vins
*@params:
* txid: current txid
* vinAccDictptr: mapping of vin txid to output accounts
* currOutAccptr: current output account pointer
*/
func mergeVins(txid string, txInputs []string, vinAccDictptr *map[string] *int, currOutAccptr *int){

	for _, vin := range txInputs {
		if(vin != "coinbase"){
			if val, ok := (*vinAccDictptr)[vin]; ok {
					*val = *currOutAccptr
			}else{
				(*vinAccDictptr)[vin]= currOutAccptr
			}
		}
	}
}
/*
*updates mapping utxo to out account pointers
*maps txid to output account pointer
*@params: 
* txid: current txid
* txOutputs : array of all utxos for current txid
* utxoAccDictptr : pointer to mapping of utxo to output acc
* txidOutAccptr : pointer to mapping of txid to output acc
*/
func updateIns (txid string, vinTxids []string, txidOutDictptr *map [string] *int, txidInDictptr *map[string] **int){
	currInAcc := 1
	currInAccptr1 := &currInAcc
	for _, vin_txid := range vinTxids {

			if (vin_txid != "coinbase"){
					if val, ok := (*txidOutDictptr)[vin_txid]; ok{
							currInAccptr1 = val

					}else{
							log.Println("somethings wrong")
					}
			}
	}
	(*txidInDictptr)[txid] = &currInAccptr1
}

/*
* formats adress into sorrect string format
*@params: adress
*@returns: adress (string)
*/
func formatString(address interface{})(string){
	inputString := fmt.Sprintf("%v", address)
	trimSuff := strings.TrimSuffix(inputString, "]")
	trimPre := strings.TrimPrefix(trimSuff, "[")
	return trimPre
}

/*
* formats input Acc, output Acc, Value into correct edgelist format
*@returns: output string "in out value" (string)
*/
func makeMMString(inAcc int, outAcc int, val uint64)(string){
	mmStr := strconv.Itoa(inAcc) + " " + strconv.Itoa(outAcc) + " " + strconv.FormatUint(val, 10)
	return mmStr
}

/*
*formats transaction value into Satoshis
*@params: float value of transaction in btc 
*@returns: value in satoshi
*/
func valueFormatter (floatVal float64)(uint64){
	s := fmt.Sprintf("%.8f", floatVal)
	f, _ := strconv.ParseFloat(s, 64)
	var i uint64 = uint64(f*100000000.0)
	return i
}

func orderAccounts(txidOutDictptr *map [string] *int, orderedAccptr *map[int] int){
	(*orderedAccptr)[1] = 0	//initialize coinbase
	i:=1
	for _, accptr := range *txidOutDictptr{
		if _, ok := (*orderedAccptr)[*accptr]; !ok {
			(*orderedAccptr)[*accptr] = i
			i++
		}

	}
	log.Println("total ", i)
}

func main(){
	utxoAccDict := map[string] *int{}
	txidOutDict := map[string] *int{}
	txidInDict := map[string] **int{}
	txidValue := map[string] uint64{}
	vinAccDict := map[string] *int{}
	orderedAccDict := map[int] int{}

	for i := 0; i<150000; i++{
        rpcgetUTXOs(i, &utxoAccDict, &txidOutDict, &txidInDict, &txidValue, &vinAccDict)
	}
	//output file
	f, err := os.Create("outEdge")
	if err != nil {
			fmt.Println(err)
			f.Close()
	return
	}
	//metadata file
	metaF, err := os.Create("metadata")
	if err != nil {
			fmt.Println(err)
			metaF.Close()
	return
	}

	orderAccounts(&txidOutDict, &orderedAccDict)
	log.Println(len(txidOutDict))
	for txid, outptr:= range txidOutDict{
		inAcc := orderedAccDict[**(txidInDict[txid])]
		outAcc := orderedAccDict[*outptr]
		strItem := makeMMString(inAcc, outAcc, txidValue[txid])
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
	}
	f.Close()
	metaF.Close()
}

