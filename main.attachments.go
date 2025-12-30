package main

import (
	"bufio"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	_ "bytes"
	_ "io"

	apiLib "github.com/hornbill/goApiLib"
	"github.com/hornbill/pb"
)

func processAttachments() {
	//Process attachments for all imported requests
	espXmlmc, err := NewEspXmlmcSession()
	if err != nil {
		logger(4, "Could not connect to Hornbill Instance: "+err.Error(), false)
		return

	}
	logger(1, "Processing file attachments for "+fmt.Sprint(len(arrCallsLogged))+" imported requests.", true)
	bar := pb.StartNew(len(arrCallsLogged))
	for swRef, smRef := range arrCallsLogged {
		processFileAttachments(swRef, smRef, espXmlmc)
		bar.Increment()
	}
	bar.FinishPrint("File Attachment Import Complete")
}

func processFileAttachments(swCallRef, smCallRef string, espXmlmc *apiLib.XmlmcInstStruct) {

	requestAttachments := fileAttachmentData(swCallRef, smCallRef)
	if len(requestAttachments) > 0 {
		logger(1, "Processing "+strconv.Itoa(len(requestAttachments))+"  File Attachments for "+swCallRef+"["+smCallRef+"]", false)
	}
	for i := 0; i < len(requestAttachments); i++ {
		entityRequest := ""
		fileRecord := requestAttachments[i]
		fileRecord.SmCallRef = smCallRef
		entityRequest = "Requests"

//		updateID, _ := strconv.Atoi(fileRecord.UpdateID)
//		updateID = updateID - 1
//		fileRecord.UpdateID = strconv.Itoa(updateID)

		fileRecord.Extension = filepath.Ext(fileRecord.FileName)

		var err error
		fileRecord.FileData, err = getFileEncoded(fileRecord)
		if err == nil {
			if (fileRecord.AddedBy != ""){
				if (fileRecord.Description == "") {
					fileRecord.Description = "Originally added by " + fileRecord.AddedBy
				} else {
					fileRecord.Description = fileRecord.Description + "\nOriginally added by " + fileRecord.AddedBy
				}
			}
			addFileContent(entityRequest, fileRecord, espXmlmc)
		}
		
	}
}

//getFileEncoded - get encoded file data
func getFileEncoded(fileRecord fileAssocStruct) (string, error) {
//	subFolderName := getSubFolderName(fileRecord.CallRef)
//	hostFileName := padCallRef(fileRecord.CallRef, "f", 8) + "." + padCallRef(fileRecord.DataID, "", 3)
	fullFilePath := swImportConf.AttachmentRoot + "/" + fileRecord.FilePath
	logger(1, "Retrieving File ["+fileRecord.FileName+"] from: "+fullFilePath, false)

	//Get File Data
	if _, fileCheckErr := os.Stat(fullFilePath); os.IsNotExist(fileCheckErr) {
		logger(4, "File does not exist at location", false)
		return "", fileCheckErr
	}
	//-- Load Config File
	file, fileError := os.Open(fullFilePath)
	//-- Check For Error Reading File
	if fileError != nil {
		logger(4, "Error Opening File: "+fileError.Error(), false)
		return "", fileError
	}
	defer file.Close()
	// create a new buffer base on file size
	fInfo, _ := file.Stat()
	size := fInfo.Size()
	buf := make([]byte, size)

	// read file content into buffer
	fReader := bufio.NewReader(file)
	fReader.Read(buf)
	fileEncoded := base64.StdEncoding.EncodeToString(buf)
	return fileEncoded, nil
}

//Get file attachment records from Supportworks
func fileAttachmentData(swRequest, smRequest string) []fileAssocStruct {
	intSwCallRef := getCallRefInt(swRequest)
	var returnArray = make([]fileAssocStruct, 0)
	//Connect to the JSON specified DB
	//Check connection is open
	err := dbsys.Ping()
	if err != nil {
		logger(4, "[DATABASE] [PING] Database Connection Error for Request File Attachments: "+err.Error(), false)
		return returnArray
	}

	
	sqlFileQuery := strings.ReplaceAll(swImportConf.AttachmentQuery, "[sourceref]", intSwCallRef)

	
	//Run Query
	rows, err := dbsys.Queryx(sqlFileQuery)
	if err != nil {
		logger(4, " Database Query Error: "+err.Error(), false)
		return returnArray
	}
	//-- Iterate through file attachment records returned from SQL query:
	for rows.Next() {
		//Scan current file attachment record in to struct
		var requestAttachment fileAssocStruct
		err = rows.StructScan(&requestAttachment)
		if err != nil {
			logger(4, " Data Mapping Error: "+err.Error(), false)
		}
		//Add to array for reponse
		returnArray = append(returnArray, requestAttachment)
	}
	return returnArray
}



//addFileContent - reads the file attachment from Supportworks, attach to request and update content location
func addFileContent(entityName string, fileRecord fileAssocStruct, espXmlmc *apiLib.XmlmcInstStruct) bool {
	logger(1, "Adding "+fileRecord.FileName, false)

	//Get rid of new line or carriage return characters from Base64 string
	rexNL := regexp.MustCompile(`\r?\n`)
	fileRecord.FileData = rexNL.ReplaceAllString(fileRecord.FileData, "")

	//If using the Requests entity, set primary key to be the SM request ref
	attPriKey := fileRecord.FileID
	if entityName == "Requests" {
		attPriKey = fileRecord.SmCallRef
	}
	filenameReplacer := strings.NewReplacer("<", "_", ">", "_", "|", "_", "\\", "_", "/", "_", ":", "_", "*", "_", "?", "_", "\"", "_")
	useFileName := filenameReplacer.Replace(fileRecord.FileName)
	if entityName == "RequestHistoricUpdateAttachments" {
		espXmlmc.SetParam("application", appServiceManager)
		espXmlmc.SetParam("entity", "RequestHistoricUpdateAttachments")
		espXmlmc.SetParam("returnModifiedData", "true")
		espXmlmc.OpenElement("primaryEntityData")
		espXmlmc.OpenElement("record")
		espXmlmc.SetParam("h_addedby", fileRecord.AddedBy)
		espXmlmc.SetParam("h_callref", fileRecord.SmCallRef)
		espXmlmc.SetParam("h_compressed", fileRecord.Compressed)
		espXmlmc.SetParam("h_dataid", fileRecord.DataID)
		espXmlmc.SetParam("h_filename", useFileName)
		espXmlmc.SetParam("h_filetime", fileRecord.FileTime)
		espXmlmc.SetParam("h_pk_fileid", attPriKey)
		espXmlmc.SetParam("h_sizec", strconv.Itoa(int(fileRecord.SizeC)))
		espXmlmc.SetParam("h_sizeu", strconv.Itoa(int(fileRecord.SizeU)))
		espXmlmc.SetParam("h_timeadded", fileRecord.TimeAdded)
		espXmlmc.SetParam("h_updateid", fileRecord.UpdateID)
		espXmlmc.CloseElement("record")
		espXmlmc.CloseElement("primaryEntityData")

		var XMLSTRING = espXmlmc.GetParam()
		if configDebug {
			logger(1, "entityAddRecord::RequestHistoricUpdateAttachments:"+XMLSTRING, false)
		}

		XMLHistAtt, xmlmcErr := espXmlmc.Invoke("data", "entityAddRecord")
		if xmlmcErr != nil {
			logger(1, "RequestHistoricUpdateAttachments entityAddRecord Failed "+fmt.Sprintf("%s", xmlmcErr), false)
			if configDebug {
				logger(1, "RequestHistoricUpdateAttachments entityAddRecord Failed File Attachment Record XML "+XMLSTRING, false)
			}
			return false
		}
		var xmlRespon xmlmcAttachmentResponse
		errXMLMC := xml.Unmarshal([]byte(XMLHistAtt), &xmlRespon)
		if errXMLMC != nil {
			logger(4, "Unable to read response from Hornbill instance for Update File Attachment Record Insertion ["+useFileName+"] ["+fileRecord.SmCallRef+"]:"+errXMLMC.Error(), false)
			if configDebug {
				logger(1, "File Attachment Record XML "+XMLSTRING, false)
			}
			return false
		}
		if xmlRespon.MethodResult != "ok" {
			logger(4, "Unable to process Update File Attachment Record Insertion ["+useFileName+"] ["+fileRecord.SmCallRef+"]: "+xmlRespon.State.ErrorRet, false)
			if configDebug {
				logger(1, "File Attachment Record XML "+XMLSTRING, false)
			}
			return false
		}
		if configDebug {
			logger(1, "Historic Update File Attachment Record Insertion Success ["+useFileName+"] ["+fileRecord.SmCallRef+"]", false)
		}
		attPriKey = xmlRespon.HistFileID
	}

	//File content read - add data to instance
	espXmlmc.SetParam("application", appServiceManager)
	espXmlmc.SetParam("entity", entityName)
	espXmlmc.SetParam("keyValue", attPriKey)
	espXmlmc.SetParam("folder", "/")
	espXmlmc.OpenElement("localFile")
	espXmlmc.SetParam("fileName", useFileName)
	espXmlmc.SetParam("fileData", fileRecord.FileData)
	espXmlmc.CloseElement("localFile")
	espXmlmc.SetParam("overwrite", "true")
	var XMLSTRINGDATA = espXmlmc.GetParam()
	if configDebug {
		logger(1, "entityAttachFile:"+XMLSTRINGDATA, false)
	}
	XMLAttach, xmlmcErr := espXmlmc.Invoke("data", "entityAttachFile")
	if xmlmcErr != nil {
		logger(4, "Could not add Attachment File Data for ["+useFileName+"] ["+fileRecord.SmCallRef+"]: "+xmlmcErr.Error(), false)
		if configDebug {
			logger(1, "File Data Record XML "+XMLSTRINGDATA, false)
		}
		return false
	}
	var xmlRespon xmlmcAttachmentResponse

	err := xml.Unmarshal([]byte(XMLAttach), &xmlRespon)
	if err != nil {
		logger(4, "Could not add Attachment File Data for ["+useFileName+"] ["+fileRecord.SmCallRef+"]: "+err.Error(), false)
		if configDebug {
			logger(1, "File Data Record XML "+XMLSTRINGDATA, false)
		}
	} else {
		if xmlRespon.MethodResult != "ok" {
			logger(4, "Could not add Attachment File Data for ["+useFileName+"] ["+fileRecord.SmCallRef+"]: "+xmlRespon.State.ErrorRet, false)
			if configDebug {
				logger(1, "File Data Record XML "+XMLSTRINGDATA, false)
			}
		} else {
			//-- If we've got a Content Location back from the API, update the file record with this
			if entityName != "RequestHistoricUpdateAttachments" {
				espXmlmc.SetParam("application", appServiceManager)
				espXmlmc.SetParam("entity", "RequestAttachments")
				espXmlmc.OpenElement("primaryEntityData")
				espXmlmc.OpenElement("record")
				espXmlmc.SetParam("h_request_id", fileRecord.SmCallRef)
				espXmlmc.SetParam("h_description", fileRecord.Description)
				espXmlmc.SetParam("h_filename", useFileName)
				espXmlmc.SetParam("h_timestamp", epochToDateTime(fileRecord.TimeAdded))
				espXmlmc.SetParam("h_visibility", "trustedGuest")
				espXmlmc.CloseElement("record")
				espXmlmc.CloseElement("primaryEntityData")
				XMLSTRINGDATA = espXmlmc.GetParam()
				if configDebug {
					logger(1, "entityAddRecord::RequestAttachments:"+XMLSTRINGDATA, false)
				}

				XMLContentLoc, xmlmcErrContent := espXmlmc.Invoke("data", "entityAddRecord")
				if xmlmcErrContent != nil {
					logger(4, "Could not update request ["+fileRecord.SmCallRef+"] with attachment ["+useFileName+"]: "+xmlmcErrContent.Error(), false)
					if configDebug {
						logger(1, "File Data Record XML "+XMLSTRINGDATA, false)
					}
					return false
				}
				var xmlResponLoc xmlmcResponse

				err = xml.Unmarshal([]byte(XMLContentLoc), &xmlResponLoc)
				if err != nil {
					logger(4, "Added file data to but unable to set Content Location on ["+fileRecord.SmCallRef+"] for File Content ["+useFileName+"] - read response from Hornbill instance:"+err.Error(), false)
					if configDebug {
						logger(1, "File Data Record XML "+XMLSTRINGDATA, false)
					}
					return false
				}
				if xmlResponLoc.MethodResult != "ok" {
					logger(4, "Added file data but unable to set Content Location on ["+fileRecord.SmCallRef+"] for File Content ["+useFileName+"]: "+xmlResponLoc.State.ErrorRet, false)
					if configDebug {
						logger(1, "File Data Record XML "+XMLSTRINGDATA, false)
					}
					return false
				}
				logger(1, entityName+" File Content ["+useFileName+"] Added to ["+fileRecord.SmCallRef+"] Successfully", false)
			}
			counters.filesAttached++
		}
	}
	return true
}


func getCallRefInt(callRef string) string {
	re1, err := regexp.Compile(`[0-9]+`)
	if err != nil {
		logger(4, "Error converting string Supportworks call reference:"+err.Error(), false)
	}
	result := strings.TrimLeft(re1.FindString(callRef), "0")
	return result
}
