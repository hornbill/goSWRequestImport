package main

import (
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"fmt"

	/* non core libraries */
	apiLib "github.com/hornbill/goApiLib"
)

// espLogger -- Log to ESP
func espLogger(message, severity string) {
	espXmlmc.SetParam("fileName", "SW_Call_Import")
	espXmlmc.SetParam("group", "general")
	espXmlmc.SetParam("severity", severity)
	espXmlmc.SetParam("message", message)
	espXmlmc.Invoke("system", "logMessage")
}

//doesUserExist takes an User ID string and returns a true if one exists in the cache or on the Instance
func doesUserExist(userID string, espXmlmc *apiLib.XmlmcInstStruct, buffer *bytes.Buffer) bool {
	boolUserExists := false
	if userID != "" {
		userInCache, userName, _ := userInCache(userID)
		//-- Check if we have cached the Analyst already
		if userInCache && userName != "" {
			boolUserExists = true
		} else {
			//Get Analyst Info
			espXmlmc.SetParam("userId", userID)

			XMLAnalystSearch, xmlmcErr := espXmlmc.Invoke("admin", "userGetInfo")
			if xmlmcErr != nil {
				buffer.WriteString(loggerGen(4, "Unable to Search for User ["+userID+"]: "+xmlmcErr.Error()))
			}

			var xmlRespon xmlmcUserListResponse
			err := xml.Unmarshal([]byte(XMLAnalystSearch), &xmlRespon)
			if err != nil {
				buffer.WriteString(loggerGen(4, "Unable to Search for User ["+userID+"]: "+err.Error()))
			} else {
				if xmlRespon.MethodResult != "ok" {
					//Analyst most likely does not exist
					buffer.WriteString(loggerGen(5, "Unable to Search for User ["+userID+"]: "+xmlRespon.State.ErrorRet))
				} else {
					//-- Check Response
					if xmlRespon.FullName != "" {
						boolUserExists = true
						//-- Add Analyst to Cache
						var userForCache userListStruct
						userForCache.UserID = userID
						userForCache.Name = xmlRespon.FullName
						userForCache.HomeOrg = xmlRespon.HomeOrg
						userNamedMap := []userListStruct{userForCache}
						mutexAnalysts.Lock()
						users = append(users, userNamedMap...)
						mutexAnalysts.Unlock()
					}
				}
			}
		}
	}
	return boolUserExists
}

//doesContactExist takes a Contact ID string and returns a true if one exists in the cache or on the Instance
func doesContactExist(contactID string, espXmlmc *apiLib.XmlmcInstStruct, buffer *bytes.Buffer) bool {
	contactExists := false

	if contactID != "" {
		customerIsInCache, contactName, _, _ := contactInCache(contactID)
		//-- Check if we have cached the Analyst already
		if customerIsInCache && contactName != "" {
			return true
		}
		//Get Analyst Info
		espXmlmc.SetParam("application", "com.hornbill.core")
		espXmlmc.SetParam("entity", "Contact")
		espXmlmc.SetParam("matchScope", "all")
		espXmlmc.OpenElement("searchFilter")
		espXmlmc.SetParam("column", "h_logon_id")
		espXmlmc.SetParam("value", contactID)
		espXmlmc.SetParam("matchType", "exact")
		espXmlmc.CloseElement("searchFilter")
		espXmlmc.SetParam("maxResults", "1")
		XMLCustomerSearch, xmlmcErr := espXmlmc.Invoke("data", "entityBrowseRecords2")
		if xmlmcErr != nil {
			buffer.WriteString(loggerGen(4, "Unable to Search for Contact ["+contactID+"]: "+xmlmcErr.Error()))
		}
		var xmlRespon xmlmcContactListResponse

		err := xml.Unmarshal([]byte(XMLCustomerSearch), &xmlRespon)
		if err != nil {
			buffer.WriteString(loggerGen(4, "Unable to Search for Contact ["+contactID+"]: "+err.Error()))
		} else {
			if xmlRespon.MethodResult != "ok" {
				//Customer most likely does not exist
				buffer.WriteString(loggerGen(5, "Unable to Search for Contact ["+contactID+"]: "+xmlRespon.State.ErrorRet))
			} else {
				//-- Check Response
				if xmlRespon.CustomerFirstName != "" {
					contactExists = true
					//-- Add Customer to Cache
					var newCustomerForCache customerListStruct
					newCustomerForCache.CustomerID = contactID
					newCustomerForCache.CustomerHornbillID = xmlRespon.CustomerHornbillID
					newCustomerForCache.CustomerOrgID = xmlRespon.CustomerOrgID
					newCustomerForCache.CustomerName = xmlRespon.CustomerFirstName + " " + xmlRespon.CustomerLastName
					customerNamedMap := []customerListStruct{newCustomerForCache}
					mutexCustomers.Lock()
					customers = append(customers, customerNamedMap...)
					mutexCustomers.Unlock()

					buffer.WriteString(loggerGen(1, "Added Contact ["+contactID+"]: "+newCustomerForCache.CustomerName))

				}
			}
		}
	}
	return contactExists
}

//NewEspXmlmcSession - New Xmlmc Session variable (Cloned Session)
func NewEspXmlmcSession() (*apiLib.XmlmcInstStruct, error) {
	espXmlmcLocal := apiLib.NewXmlmcInstance(swImportConf.HBConf.InstanceID)
	if swImportConf.HBConf.APIKey != "" {
		espXmlmcLocal.SetAPIKey(swImportConf.HBConf.APIKey)
	} else {
		espXmlmcLocal.SetSessionID(espXmlmc.GetSessionID())
	}
	return espXmlmcLocal, nil
}

//-- start ESP user session
func login() bool {
	logger(1, "Logging Into: "+swImportConf.HBConf.InstanceID, true)
	logger(1, "UserName: "+swImportConf.HBConf.UserName, false)
	espXmlmc = apiLib.NewXmlmcInstance(swImportConf.HBConf.InstanceID)

	espXmlmc.SetParam("userId", swImportConf.HBConf.UserName)
	espXmlmc.SetParam("password", base64.StdEncoding.EncodeToString([]byte(swImportConf.HBConf.Password)))
	XMLLogin, xmlmcErr := espXmlmc.Invoke("session", "userLogon")
	if xmlmcErr != nil {
		logger(4, "Unable to Login: "+xmlmcErr.Error(), true)
		return false
	}

	var xmlRespon xmlmcResponse
	err := xml.Unmarshal([]byte(XMLLogin), &xmlRespon)
	if err != nil {
		logger(4, "Unable to Login: "+err.Error(), true)
		return false
	}
	if xmlRespon.MethodResult != "ok" {
		logger(4, "Unable to Login: "+xmlRespon.State.ErrorRet, true)
		return false
	}
	espLogger("---- Supportworks Call Import Utility V"+fmt.Sprintf("%v", version)+" ----", "debug")
	espLogger("Logged In As: "+swImportConf.HBConf.UserName, "debug")
	return true
}

//logout -- XMLMC Logout
//-- Adds details to log file, ends user ESP session
func logout() {
	//-- End output
	espLogger("Requests Logged: "+fmt.Sprintf("%d", counters.created), "debug")
	espLogger("Requests Skipped: "+fmt.Sprintf("%d", counters.createdSkipped), "debug")
	espLogger("Time Taken: "+fmt.Sprintf("%v", endTime), "debug")
	espLogger("---- Supportworks Call Import Complete ---- ", "debug")
	logger(1, "Logout", true)
	espXmlmc.Invoke("session", "userLogoff")
}
