package main

import (
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
)

//getCallPriorityID takes the Call Record and returns a correct Priority ID if one exists on the Instance
func getCallPriorityID(strPriorityName string) (string, string) {
	priorityID := ""
	if swImportConf.PriorityMapping[strPriorityName] != nil {
		strPriorityName = fmt.Sprintf("%s", swImportConf.PriorityMapping[strPriorityName])
		if strPriorityName != "" {
			priorityID = getPriorityID(strPriorityName)
		}
	}
	return priorityID, strPriorityName
}

//getPriorityID takes a Priority Name string and returns a correct Priority ID if one exists in the cache or on the Instance
func getPriorityID(priorityName string) string {
	priorityID := ""
	if priorityName != "" {
		priorityIsInCache, PriorityIDCache := recordInCache(priorityName, "Priority")
		//-- Check if we have cached the Priority already
		if priorityIsInCache {
			priorityID = PriorityIDCache
		} else {
			priorityIsOnInstance, PriorityIDInstance := searchPriority(priorityName)
			//-- If Returned set output
			if priorityIsOnInstance {
				priorityID = strconv.Itoa(PriorityIDInstance)
			}
		}
	}
	return priorityID
}

// seachPriority -- Function to check if passed-through priority name is on the instance
func searchPriority(priorityName string) (bool, int) {
	boolReturn := false
	intReturn := 0
	//-- ESP Query for Priority
	espXmlmc, err := NewEspXmlmcSession()
	if err != nil {
		return false, 0
	}
	//-- ESP Query for Priority
	espXmlmc.SetParam("application", appServiceManager)
	espXmlmc.SetParam("entity", "Priority")
	espXmlmc.SetParam("matchScope", "all")
	espXmlmc.OpenElement("searchFilter")
	//espXmlmc.SetParam("h_priorityname", priorityName)
	espXmlmc.SetParam("column", "h_priorityname")
	espXmlmc.SetParam("value", priorityName)
	espXmlmc.CloseElement("searchFilter")
	espXmlmc.SetParam("maxResults", "1")

	XMLPrioritySearch, xmlmcErr := espXmlmc.Invoke("data", "entityBrowseRecords2")
	if xmlmcErr != nil {
		logger(4, "Unable to Search for Priority: "+fmt.Sprintf("%v", xmlmcErr), false)
		return boolReturn, intReturn
		//log.Fatal(xmlmcErr)
	}
	var xmlRespon xmlmcPriorityListResponse

	err = xml.Unmarshal([]byte(XMLPrioritySearch), &xmlRespon)
	if err != nil {
		logger(4, "Unable to Search for Priority: "+fmt.Sprintf("%v", err), false)
	} else {
		if xmlRespon.MethodResult != "ok" {
			logger(5, "Unable to Search for Priority: "+xmlRespon.State.ErrorRet, false)
		} else {
			//-- Check Response
			if xmlRespon.PriorityName != "" {
				if strings.ToLower(xmlRespon.PriorityName) == strings.ToLower(priorityName) {
					intReturn = xmlRespon.PriorityID
					boolReturn = true
					//-- Add Priority to Cache
					var newPriorityForCache priorityListStruct
					newPriorityForCache.PriorityID = intReturn
					newPriorityForCache.PriorityName = priorityName
					priorityNamedMap := []priorityListStruct{newPriorityForCache}
					mutexPriorities.Lock()
					priorities = append(priorities, priorityNamedMap...)
					mutexPriorities.Unlock()
				}
			}
		}
	}
	return boolReturn, intReturn
}
