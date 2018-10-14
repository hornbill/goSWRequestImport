package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/hornbill/goApiLib"
)

//getCallCategoryID takes the Call Record and returns a correct Category ID if one exists on the Instance
func getCallCategoryID(callMap map[string]interface{}, categoryGroup string, espXmlmc *apiLib.XmlmcInstStruct, buffer *bytes.Buffer) (string, string) {
	categoryID := ""
	categoryString := ""
	categoryNameMapping := ""
	categoryCode := ""
	if categoryGroup == "Request" {
		categoryNameMapping = fmt.Sprintf("%v", mapGenericConf.CoreFieldMapping["h_category_id"])
		categoryCode = getFieldValue(categoryNameMapping, callMap)
		if swImportConf.CategoryMapping[categoryCode] != nil {
			//Get Category Code from JSON mapping
			categoryCode = fmt.Sprintf("%s", swImportConf.CategoryMapping[categoryCode])
		} else {
			//Mapping doesn't exist - replace hyphens from SW Profile code with another string, and try to use this
			//SMProfileCodeSeperator allows us to specify in the config, the seperator used within Service Manager
			//profile codes
			categoryCode = strings.Replace(categoryCode, "-", swImportConf.SMProfileCodeSeperator, -1)
		}

	} else {
		categoryNameMapping = fmt.Sprintf("%v", mapGenericConf.CoreFieldMapping["h_closure_category_id"])
		categoryCode = getFieldValue(categoryNameMapping, callMap)
		if swImportConf.ResolutionCategoryMapping[categoryCode] != nil {
			//Get Category Code from JSON mapping
			categoryCode = fmt.Sprintf("%s", swImportConf.ResolutionCategoryMapping[categoryCode])
		} else {
			//Mapping doesn't exist - replace hyphens from SW Profile code with colon, and try to use this
			categoryCode = strings.Replace(categoryCode, "-", swImportConf.SMProfileCodeSeperator, -1)
		}
	}
	if categoryCode != "" {
		categoryID, categoryString = getCategoryID(categoryCode, categoryGroup, espXmlmc, buffer)
	}
	return categoryID, categoryString
}

//getCategoryID takes a Category Code string and returns a correct Category ID if one exists in the cache or on the Instance
func getCategoryID(categoryCode, categoryGroup string, espXmlmc *apiLib.XmlmcInstStruct, buffer *bytes.Buffer) (string, string) {

	categoryID := ""
	categoryString := ""
	if categoryCode != "" {
		categoryIsInCache, CategoryIDCache, CategoryNameCache := categoryInCache(categoryCode, categoryGroup+"Category")
		//-- Check if we have cached the Category already
		if categoryIsInCache {
			categoryID = CategoryIDCache
			categoryString = CategoryNameCache
		} else {
			categoryIsOnInstance, CategoryIDInstance, CategoryStringInstance := searchCategory(categoryCode, categoryGroup, espXmlmc, buffer)
			//-- If Returned set output
			if categoryIsOnInstance {
				categoryID = CategoryIDInstance
				categoryString = CategoryStringInstance
			}
		}
	}
	return categoryID, categoryString
}

// seachCategory -- Function to check if passed-through support category name is on the instance
func searchCategory(categoryCode, categoryGroup string, espXmlmc *apiLib.XmlmcInstStruct, buffer *bytes.Buffer) (bool, string, string) {
	boolReturn := false
	idReturn := ""
	strReturn := ""
	//-- ESP Query for category
	espXmlmc.SetParam("codeGroup", categoryGroup)
	espXmlmc.SetParam("code", categoryCode)
	XMLCategorySearch, xmlmcErr := espXmlmc.Invoke("data", "profileCodeLookup")
	if xmlmcErr != nil {
		buffer.WriteString(loggerGen(4, "XMLMC API Invoke Failed for "+categoryGroup+" Category ["+categoryCode+"]: "+fmt.Sprintf("%v", xmlmcErr)))
		return boolReturn, idReturn, strReturn
	}
	var xmlRespon xmlmcCategoryListResponse

	err := xml.Unmarshal([]byte(XMLCategorySearch), &xmlRespon)
	if err != nil {
		buffer.WriteString(loggerGen(4, "Unable to unmarshal response for "+categoryGroup+" Category: "+fmt.Sprintf("%v", err)))
	} else {
		if xmlRespon.MethodResult != "ok" {
			buffer.WriteString(loggerGen(5, "Unable to Search for "+categoryGroup+" Category ["+categoryCode+"]: ["+fmt.Sprintf("%v", xmlRespon.MethodResult)+"] "+xmlRespon.State.ErrorRet))
		} else {
			//-- Check Response
			if xmlRespon.CategoryName != "" {
				strReturn = xmlRespon.CategoryName
				idReturn = xmlRespon.CategoryID
				buffer.WriteString(loggerGen(3, "[CATEGORY] [SUCCESS] Methodcall result OK for "+categoryGroup+" Category ["+categoryCode+"] : ["+strReturn+"]"))
				boolReturn = true
				//-- Add Category to Cache
				var newCategoryForCache categoryListStruct
				newCategoryForCache.CategoryID = idReturn
				newCategoryForCache.CategoryCode = categoryCode
				newCategoryForCache.CategoryName = strReturn
				categoryNamedMap := []categoryListStruct{newCategoryForCache}
				switch categoryGroup {
				case "Request":
					mutexCategories.Lock()
					categories = append(categories, categoryNamedMap...)
					mutexCategories.Unlock()
				case "Closure":
					mutexCloseCategories.Lock()
					closeCategories = append(closeCategories, categoryNamedMap...)
					mutexCloseCategories.Unlock()
				}
			} else {
				buffer.WriteString(loggerGen(5, "[CATEGORY] Methodcall result OK for "+categoryGroup+" Category ["+categoryCode+"] but category name blank: ["+xmlRespon.CategoryID+"] ["+xmlRespon.CategoryName+"]"))
			}
		}
	}
	return boolReturn, idReturn, strReturn
}
