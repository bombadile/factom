// Copyright 2016 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package wsapi

import (
	"encoding/json"
	"io/ioutil"

	"github.com/FactomProject/factom"
	"github.com/FactomProject/factom/wallet"
	"github.com/FactomProject/web"
)

const APIVersion string = "2.0"

var (
	webServer *web.Server
	fctWallet *wallet.Wallet
)

func Start(w *wallet.Wallet, net string) {
	webServer = web.NewServer()
	fctWallet = w

	webServer.Post("/v2", handleV2)
	webServer.Get("/v2", handleV2)
	webServer.Run(net)
}

func Stop() {
	fctWallet.Close()
	webServer.Close()
}

func handleV2(ctx *web.Context) {
	body, err := ioutil.ReadAll(ctx.Request.Body)
	if err != nil {
		handleV2Error(ctx, nil, newInvalidRequestError())
		return
	}

	j, err := factom.ParseJSON2Request(string(body))
	if err != nil {
		handleV2Error(ctx, nil, newInvalidRequestError())
		return
	}

	jsonResp, jsonError := handleV2Request(j)

	if jsonError != nil {
		handleV2Error(ctx, j, jsonError)
		return
	}

	ctx.Write([]byte(jsonResp.String()))
}

func handleV2Request(j *factom.JSON2Request) (*factom.JSON2Response, *factom.JSONError) {
	var resp interface{}
	var jsonError *factom.JSONError
	params := j.Params

	switch j.Method {
	case "address":
		resp, jsonError = handleAddress(params)
	case "all-addresses":
		resp, jsonError = handleAllAddresses(params)
	case "generate-ec-address":
		resp, jsonError = handleGenerateECAddress(params)
	case "generate-factoid-address":
		resp, jsonError = handleGenerateFactoidAddress(params)
	case "import-addresses":
		resp, jsonError = handleImportAddresses(params)
	case "wallet-backup":
		resp, jsonError = handleWalletBackup(params)
	case "new-transaction":
		resp, jsonError = handleNewTransaction(params)
	case "delete-transaction":
		resp, jsonError = handleDeleteTransaction(params)
	case "add-input":
		resp, jsonError = handleAddInput(params)
	case "add-output":
		resp, jsonError = handleAddOutput(params)
	case "add-ec-output":
		resp, jsonError = handleAddECOutput(params)
	case "add-fee":
		resp, jsonError = handleAddFee(params)
	case "sub-fee":
		resp, jsonError = handleSubFee(params)
	case "sign-transaction":
		resp, jsonError = handleSignTransaction(params)
	case "compose-transaction":
		resp, jsonError = handleComposeTransaction(params)
	default:
		jsonError = newMethodNotFoundError()
	}
	if jsonError != nil {
		return nil, jsonError
	}

	jsonResp := factom.NewJSON2Response()
	jsonResp.ID = j.ID
	jsonResp.Result = resp

	return jsonResp, nil
}

func handleAddress(params interface{}) (interface{}, *factom.JSONError) {
	req := new(addressRequest)
	if err := mapToObject(params, req); err != nil {
		return nil, newInvalidParamsError()
	}

	resp := new(addressResponse)
	switch factom.AddressStringType(req.Address) {
	case factom.ECPub:
		e, err := fctWallet.GetECAddress(req.Address)
		if err != nil {
			return nil, newCustomInternalError(err)
		}
		resp = mkAddressResponse(e)
	case factom.FactoidPub:
		f, err := fctWallet.GetFCTAddress(req.Address)
		if err != nil {
			return nil, newCustomInternalError(err)
		}
		resp = mkAddressResponse(f)
	default:
		return nil, newCustomInternalError("Invalid address type")
	}

	return resp, nil
}

func handleAllAddresses(params interface{}) (interface{}, *factom.JSONError) {
	resp := new(multiAddressResponse)

	fs, es, err := fctWallet.GetAllAddresses()
	if err != nil {
		return nil, newCustomInternalError(err)
	}
	for _, f := range fs {
		a := mkAddressResponse(f)
		resp.Addresses = append(resp.Addresses, a)
	}
	for _, e := range es {
		a := mkAddressResponse(e)
		resp.Addresses = append(resp.Addresses, a)
	}

	return resp, nil
}

func handleGenerateFactoidAddress(params interface{}) (interface{}, *factom.JSONError) {
	a, err := fctWallet.GenerateFCTAddress()
	if err != nil {
		return nil, newCustomInternalError(err)
	}
	
	resp := mkAddressResponse(a)
	
	return resp, nil
}

func handleGenerateECAddress(params interface{}) (interface{}, *factom.JSONError) {
	a, err := fctWallet.GenerateECAddress()
	if err != nil {
		return nil, newCustomInternalError(err)
	}
	
	resp := mkAddressResponse(a)
	
	return resp, nil
}

func handleImportAddresses(params interface{})  (interface{}, *factom.JSONError) {
	req := new(importRequest)
	if err := mapToObject(params, req); err != nil {
		return nil, newInvalidParamsError()
	}
	
	resp := new(multiAddressResponse)
	for _, v := range req.Addresses {
		switch factom.AddressStringType(v.Secret) {
		case factom.FactoidSec:
			f, err := factom.GetFactoidAddress(v.Secret)
			if err != nil {
				return nil, newCustomInternalError(err)
			}
			if err := fctWallet.PutFCTAddress(f); err != nil {
				return nil, newCustomInternalError(err)
			}
			a := mkAddressResponse(f)
			resp.Addresses = append(resp.Addresses, a)
		case factom.ECSec:
			e, err := factom.GetECAddress(v.Secret)
			if err != nil {
				return nil, newCustomInternalError(err)
			}
			if err := fctWallet.PutECAddress(e); err != nil {
				return nil, newCustomInternalError(err)
			}
			a := mkAddressResponse(e)
			resp.Addresses = append(resp.Addresses, a)
		default:
			return nil, newCustomInternalError("address could not be imported")
		}
	}
	return resp, nil
}

func handleWalletBackup(params interface{}) (interface{}, *factom.JSONError) {
	resp := new(walletBackupResponse)

	if seed, err := fctWallet.GetSeed(); err != nil {
		return nil, newCustomInternalError(err)
	} else {
		resp.Seed = seed
	}
	
	fs, es, err := fctWallet.GetAllAddresses()
	if err != nil {
		return nil, newCustomInternalError(err)
	}
	for _, f := range fs {
		a := mkAddressResponse(f)
		resp.Addresses = append(resp.Addresses, a)
	}
	for _, e := range es {
		a := mkAddressResponse(e)
		resp.Addresses = append(resp.Addresses, a)
	}

	return resp, nil
}

// transaction handlers

func handleNewTransaction(params interface{}) (interface{}, *factom.JSONError) {
	req := new(transactionRequest)
	if err := mapToObject(params, req); err != nil {
		return nil, newInvalidParamsError()
	}
	
	if err := fctWallet.NewTransaction(req.Name); err != nil {
		return nil, newCustomInternalError(err)
	}
	return "success", nil
}

func handleDeleteTransaction(params interface{}) (interface{}, *factom.JSONError) {
	req := new(transactionRequest)
	if err := mapToObject(params, req); err != nil {
		return nil, newInvalidParamsError()
	}
	
	if err := fctWallet.DeleteTransaction(req.Name); err != nil {
		return nil, newCustomInternalError(err)
	}
	return "success", nil
}

func handleAddInput(params interface{}) (interface{}, *factom.JSONError) {
	req := new(transactionValueRequest)
	if err := mapToObject(params, req); err != nil {
		return nil, newInvalidParamsError()
	}
	
	if err := fctWallet.AddInput(req.Name, req.Address, req.Amount); err != nil {
		return nil, newCustomInternalError(err)
	}
	return "success", nil
}

func handleAddOutput(params interface{}) (interface{}, *factom.JSONError) {
	req := new(transactionValueRequest)
	if err := mapToObject(params, req); err != nil {
		return nil, newInvalidParamsError()
	}
	
	if err := fctWallet.AddOutput(req.Name, req.Address, req.Amount); err != nil {
		return nil, newCustomInternalError(err)
	}
	return "success", nil
}

func handleAddECOutput(params interface{}) (interface{}, *factom.JSONError) {
	req := new(transactionValueRequest)
	if err := mapToObject(params, req); err != nil {
		return nil, newInvalidParamsError()
	}
	
	if err := fctWallet.AddECOutput(req.Name, req.Address, req.Amount); err != nil {
		return nil, newCustomInternalError(err)
	}
	return "success", nil
}

func handleAddFee(params interface{}) (interface{}, *factom.JSONError) {
	req := new(transactionAddressRequest)
	if err := mapToObject(params, req); err != nil {
		return nil, newInvalidParamsError()
	}
	
	rate, err := factom.GetRate()
	if err != nil {
		return nil, newCustomInternalError(err)
	}
	if err := fctWallet.AddFee(req.Name, req.Address, rate); err != nil {
		return nil, newCustomInternalError(err)
	}
	return "success", nil
}

func handleSubFee(params interface{}) (interface{}, *factom.JSONError) {
	req := new(transactionAddressRequest)
	if err := mapToObject(params, req); err != nil {
		return nil, newInvalidParamsError()
	}
	
	rate, err := factom.GetRate()
	if err != nil {
		return nil, newCustomInternalError(err)
	}
	if err := fctWallet.SubFee(req.Name, req.Address, rate); err != nil {
		return nil, newCustomInternalError(err)
	}
	return "success", nil
}

func handleSignTransaction(params interface{}) (interface{}, *factom.JSONError) {
	req := new(transactionRequest)
	if err := mapToObject(params, req); err != nil {
		return nil, newInvalidParamsError()
	}
	
	if err := fctWallet.SignTransaction(req.Name); err != nil {
		return nil, newCustomInternalError(err)
	}
	return "success", nil
}

func handleComposeTransaction(params interface{}) (interface{}, *factom.JSONError) {
	req := new(transactionRequest)
	if err := mapToObject(params, req); err != nil {
		return nil, newInvalidParamsError()
	}
	
	t, err := fctWallet.ComposeTransaction(req.Name)
	if err != nil {
		return nil, newCustomInternalError(err)
	}
	return t, nil
}

// utility functions

type addressResponder interface {
	PubString() string
	SecString() string
}

func mkAddressResponse(a addressResponder) *addressResponse {
	r := new(addressResponse)
	r.Public = a.PubString()
	r.Secret = a.SecString()
	return r
}

func mapToObject(source interface{}, dst interface{}) error {
	b, err := json.Marshal(source)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, dst)
}