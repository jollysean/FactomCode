package main

import (
	"github.com/FactomProject/FactomCode/wallet"
	"bytes"
	"fmt"
	"github.com/FactomProject/gocoding"
	"github.com/hoisie/web"
	"github.com/FactomProject/FactomCode/notaryapi"
	"github.com/FactomProject/FactomCode/factomapi"
	"github.com/FactomProject/factom"
	"net/url"
	"strconv"
	"encoding/base64"
	"time"
  
)

var server = web.NewServer()

func serve_init() {
	
	server.Post(`/v1/submitentry/?`, handleSubmitEntry)
	server.Post(`/v1/addchain/?`, handleChainPost)	
	server.Post(`/v1/buycredit/?`, handleBuyCreditPost)		
	server.Post(`/v1/creditbalance/?`, handleGetCreditBalancePost)			
	
	server.Get(`/v1/dblocksbyrange/([^/]+)(?:/([^/]+))?`, handleDBlocksByRange)
	server.Get(`/v1/dblock/([^/]+)(?)`, handleDBlockByHash)	
	server.Get(`/v1/eblock/([^/]+)(?)`, handleEBlockByHash)	
	server.Get(`/v1/eblockbymr/([^/]+)(?)`, handleEBlockByMR)		
	server.Get(`/v1/entry/([^/]+)(?)`, handleEntryByHash)	

} 

func handleSubmitEntry(ctx *web.Context) {
	// convert a json post to a factom.Entry then submit the entry to factom
	switch ctx.Params["format"] {
	case "json":
		j := []byte(ctx.Params["entry"])
		e := new(factom.Entry)
		e.UnmarshalJSON(j)
		if err := factom.CommitEntry(e); err != nil {
			fmt.Fprintln(ctx,
				"there was a problem with submitting the entry:", err)
		}
		time.Sleep(1 * time.Second)
		if err := factom.RevealEntry(e); err != nil {
			fmt.Fprintln(ctx,
				"there was a problem with submitting the entry:", err)
		}
		fmt.Fprintln(ctx, "Entry Submitted")
	default:
		ctx.WriteHeader(403)
	}
}

//func handleEntryPost(ctx *web.Context) {
//	var abortMessage, abortReturn string
//	
//	defer func() {
//		if abortMessage != "" && abortReturn != "" {
//			ctx.Header().Add("Location", fmt.Sprint("/failed?message=", abortMessage, "&return=", abortReturn))
//			ctx.WriteHeader(303)
//		} else if abortReturn != "" {
//			ctx.Header().Add("Location", abortReturn)
//			ctx.WriteHeader(303)
//		}
//	}()
//	
//	entry := new (notaryapi.Entry)
//	reader := gocoding.ReadBytes([]byte(ctx.Params["entry"]))
//	err := factomapi.SafeUnmarshal(reader, entry)
//
//	err = factomapi.RevealEntry(1, entry)
//		
//	if err != nil {
//		abortMessage = fmt.Sprint("An error occured while submitting the entry (entry may have been accepted by the server but was not locally flagged as such): ", err.Error())
//		return
//	}
//		
//}
func handleBuyCreditPost(ctx *web.Context) {
	var abortMessage, abortReturn string
	
	defer func() {
		if abortMessage != "" && abortReturn != "" {
			ctx.Header().Add("Location", fmt.Sprint("/failed?message=", abortMessage, "&return=", abortReturn))
			ctx.WriteHeader(303)
		} 
	}()

	
	ecPubKey := new (notaryapi.Hash)
	if ctx.Params["to"] == "wallet" {
		ecPubKey.Bytes = (*wallet.ClientPublicKey().Key)[:]
	} else {
		ecPubKey.Bytes, _ = base64.URLEncoding.DecodeString(ctx.Params["to"])
	}

	fmt.Println("handleBuyCreditPost using pubkey: ", ecPubKey, " requested",ctx.Params["to"])

	factoid, _ := strconv.ParseFloat(ctx.Params["value"], 10)
	value := uint64(factoid*1000000000)
	err := factomapi.BuyEntryCredit(1, ecPubKey, nil, value, 0, nil)

		
	if err != nil {
		abortMessage = fmt.Sprint("An error occured while submitting the buycredit request: ", err.Error())
		return
	}
		 
}
func handleGetCreditBalancePost(ctx *web.Context) {	
	var httpcode int = 200
	buf := new(bytes.Buffer)

	defer func() {
		ctx.WriteHeader(httpcode)
		ctx.Write(buf.Bytes())
	}()
	
	ecPubKey := new (notaryapi.Hash)
	if ctx.Params["pubkey"] == "wallet" {
		ecPubKey.Bytes = (*wallet.ClientPublicKey().Key)[:]
	} else {
		ecPubKey.Bytes, _ = base64.StdEncoding.DecodeString(ctx.Params["pubkey"])
	}

	fmt.Println("handleGetCreditBalancePost using pubkey: ", ecPubKey, " requested",ctx.Params["pubkey"])
	
	balance, err := factomapi.GetEntryCreditBalance(ecPubKey)
	
	ecBalance := new(notaryapi.ECBalance)
	ecBalance.Credits = balance
	ecBalance.PublicKey = ecPubKey

	fmt.Println("Balance for pubkey ", ctx.Params["pubkey"], " is: ", balance)
	
	// Send back JSON response
	err = factomapi.SafeMarshal(buf, ecBalance)
	if err != nil{
		httpcode = 400
		buf.WriteString("Bad request ")
		return		
	}			
}

 
func handleChainPost(ctx *web.Context) {
	var abortMessage, abortReturn string
	defer func() {
		if abortMessage != "" && abortReturn != "" {
			ctx.Header().Add("Location", fmt.Sprint("/failed?message=", abortMessage, "&return=", abortReturn))
			ctx.WriteHeader(303)
		}
	}()
	
	fmt.Println("In handlechainPost")	
	chain := new (notaryapi.EChain)
	reader := gocoding.ReadBytes([]byte(ctx.Params["chain"]))
	err := factomapi.SafeUnmarshal(reader, chain)

	err = factomapi.RevealChain(1, chain, nil)
		
	if err != nil {
		abortMessage = fmt.Sprint("An error occured while adding the chain ", err.Error())
		return
	}
	
		 
}

func handleDBlocksByRange(ctx *web.Context, fromHeightStr string, toHeightStr string) {
	var httpcode int = 200
	buf := new(bytes.Buffer)

	defer func() {
		ctx.WriteHeader(httpcode)
		ctx.Write(buf.Bytes())
	}()
	
	fromBlockHeight, err := strconv.Atoi(fromHeightStr)
	if err != nil{
		httpcode = 400
		buf.WriteString("Bad fromBlockHeight")
		return
	}
	toBlockHeight, err := strconv.Atoi(toHeightStr)
	if err != nil{
		httpcode = 400
		buf.WriteString("Bad toBlockHeight")
		return		
	}	
	
	dBlocks, err := factomapi.GetDirectoryBloks(uint64(fromBlockHeight), uint64(toBlockHeight))
	if err != nil{
		httpcode = 400
		buf.WriteString("Bad request")
		return		
	}	

	// Send back JSON response
	err = factomapi.SafeMarshal(buf, dBlocks)
	if err != nil{
		httpcode = 400
		buf.WriteString("Bad request")
		return		
	}	
	
}


func handleDBlockByHash(ctx *web.Context, hashStr string) {
	var httpcode int = 200
	buf := new(bytes.Buffer)

	defer func() {
		ctx.WriteHeader(httpcode)
		ctx.Write(buf.Bytes())
	}()
	
	dBlock, err := factomapi.GetDirectoryBlokByHashStr(hashStr)
	if err != nil{
		httpcode = 400
		buf.WriteString("Bad Request")
		return
	}

	// Send back JSON response
	err = factomapi.SafeMarshal(buf, dBlock)
	if err != nil{
		httpcode = 400
		buf.WriteString("Bad request ")
		return		
	}	
	
}

func handleEBlockByHash(ctx *web.Context, hashStr string) {
	var httpcode int = 200
	buf := new(bytes.Buffer)

	defer func() {
		ctx.WriteHeader(httpcode)
		ctx.Write(buf.Bytes())
	}()
	
	eBlock, err := factomapi.GetEntryBlokByHashStr(hashStr)
	if err != nil{
		httpcode = 400
		buf.WriteString("Bad Request")
		return
	}

	// Send back JSON response
	err = factomapi.SafeMarshal(buf, eBlock)
	if err != nil{
		httpcode = 400
		buf.WriteString("Bad request")
		return		
	}	
	
} 
func handleEBlockByMR(ctx *web.Context, mrStr string) {
	var httpcode int = 200
	buf := new(bytes.Buffer)

	defer func() {
		ctx.WriteHeader(httpcode)
		ctx.Write(buf.Bytes())
	}()
	fmt.Println("mrstr:", mrStr)
	newstr,_ := url.QueryUnescape(mrStr)
	fmt.Println("newstr:", newstr)
	eBlock, err := factomapi.GetEntryBlokByMRStr(newstr)
	if err != nil{
		httpcode = 400
		buf.WriteString("Bad Request")
		return
	}

	// Send back JSON response
	err = factomapi.SafeMarshal(buf, eBlock)
	if err != nil{
		httpcode = 400
		buf.WriteString("Bad request")
		return		
	}	
	
} 

func handleEntryByHash(ctx *web.Context, hashStr string) {
	var httpcode int = 200
	buf := new(bytes.Buffer)

	defer func() {
		ctx.WriteHeader(httpcode)
		ctx.Write(buf.Bytes())
	}()
	
	entry, err := factomapi.GetEntryByHashStr(hashStr)
	if err != nil{
		httpcode = 400
		buf.WriteString("Bad Request")
		return
	}

	// Send back JSON response
	err = factomapi.SafeMarshal(buf, entry)
	if err != nil{
		httpcode = 400
		buf.WriteString("Bad request")
		return		
	}		
}
