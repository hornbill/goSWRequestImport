### Supportworks Call Import [GO](https://golang.org/) - Import Script to Hornbill

### Quick links
- [Overview](#overview)
- [Installation](#Installation)
- [Configuration](Cconfiguration)
    - [HBConfig](#HBConfig)
    - [Supportworks Server Address](#SWServerAddress)
    - [Attachment Root](#AttachmentRoot)
    - [System Database Configuration](#SWSystemDBConf)
    - [Application Database Configuration](#SWAppDBConf)
    - [Call Class Specific Configuration](#ConfCallClass)
    - [Priority Mapping](#PriorityMapping)
    - [Team/Support Group Mapping](#TeamMapping)
    - [Category Mapping](#CategoryMapping)
    - [Resolution Category Mapping](#ResolutionCategoryMapping)
    - [Service Mapping](#ServiceMapping)
- [Execute](#execute)
- [Testing](testing)
- [Logging](#logging)
- [Error Codes](#error codes)

# Overview
This tool provides functionality to allow the import of call data from a Supportworks 7.x or 8.x instance in to Hornbill Service Manager.

The following tasks are carried out when the tool is executed:
* Supportworks call data is extracted as per your specification, as outlined in the Configuration section of this document;
* New requests are raised on Service Manager using the extracted call data and associated mapping specifications;
* Supportworks call diary entries are imported as Historic Updates against the new Service Manager Requests;
* Attachments to Supportworks Call Diary Entries are imported against their appropriate Historic Updates within Service Manager;
* Call attachments that are not related to Call Diary Entries are attached to the relevant Service Manager request;
* Call attachments of type SWM (Supportworks Mail) are decoded and stored as plain text attachments against the Service Manager request or Historic Update as appropriate.

#### IMPORTANT!
Importing Supportworks call data and associated file attachments will consume your subscribed Hornbill storage. Please check your Administration console and your Supportworks data to ensure that you have enough subscribed storage available before running this import.

When running the import tool, after the call records are imported, you will receive a warning before importing the associated call file attachments. Please take note of the information presented, as this will inform you the amount of Hornbill storage space you have available to your instance, and the approximate amount that will be consumed should you continue with the file attachment import.

# Installation

#### Windows
* Download the archive containing the import executable
* Extract zip into a folder you would like the application to run from e.g. `C:\sw_call_import\`
* Open '''conf.json''' and add in the necessary configuration
* Open Command Line Prompt as Administrator
* Change Directory to the folder with sw_call_import.exe `C:\sw_call_import\`
* Run the command sw_call_import.exe -dryrun=true

# Configuration

Example JSON File:

```json
{
  "HBConf": {
    "UserName": "",
    "Password": "",
    "InstanceID": ""
  },
  "SWServerAddress":"127.0.0.1",
  "AttachmentRoot":"E:/Program Files/Hornbill/Supportworks Server/data/cfa_store",
  "SWSystemDBConf": {
    "Driver":"swsql",
    "UserName": "",
    "Password": ""
  },
  "SWAppDBConf": {
    "Driver": "swsql",
    "Server": "127.0.0.1",
    "Database": "swdata",
    "UserName": "",
    "Password": "",
    "Port": 5002,
    "Encrypt": false
  },
  "ConfIncident": {
    "Import":true,
    "CallClass": "Incident",
    "DefaultTeam":"",
    "DefaultPriority":"",
    "DefaultService":"",
    "SQLStatement":"SELECT opencall.callref, cust_id, itsm_title, owner, suppgroup, status, updatedb.updatetxt, priority, itsm_impact_level, itsm_urgency_level, withinfix, withinresp, bpm_workflow_id, probcode, fixcode, site FROM opencall, updatedb WHERE updatedb.callref = opencall.callref AND updatedb.udindex = 0 AND callclass = 'Incident' AND status != 17 AND appcode = 'ITSM' ",
    "CoreFieldMapping": {
      "summary":"[itsm_title]",
      "description":"Supportworks Incident Reference: [oldCallRef]\n\n[updatetxt]",
      "customerId":"[cust_id]",
      "customerType":"0",
      "ownerId":"[owner]",
      "teamId":"[suppgroup]",
      "status":"[status]",
      "priorityId":"[priority]",
      "categoryId":"[probcode]",
      "impact":"[itsm_impact_level]",
      "urgency":"[itsm_urgency_level]",
      "serviceId":"",
      "site":"[site]"
    },
    "AdditionalFieldMapping":{
      "h_withinfix":"[withinfix]",
      "h_withinresponse":"[withinresp]",
      "h_request_language":"en-GB",
      "h_external_ref_number":"[oldCallRef]",
      "h_closure_category_id":"[fixcode]",
      "h_custom_a":"",
      "h_custom_b":"",
      "h_custom_c":"",
      "h_custom_d":"",
      "h_custom_e":"",
      "h_custom_f":"",
      "h_custom_g":"",
      "h_custom_h":"",
      "h_custom_i":"",
      "h_custom_j":"",
      "h_custom_k":"",
      "h_custom_l":"",
      "h_custom_m":"",
      "h_custom_n":"",
      "h_custom_o":"",
      "h_custom_p":"",
      "h_custom_q":""
    }
  },
  "ConfServiceRequest": {
    "Import":true,
    "CallClass": "Service Request",
    "DefaultTeam":"",
    "DefaultPriority":"",
    "DefaultService":"",
    "SQLStatement":"SELECT opencall.callref, cust_id, itsm_title, owner, suppgroup, status, updatedb.updatetxt, priority, itsm_impact_level, itsm_urgency_level, withinfix, withinresp, bpm_workflow_id, probcode, fixcode, site, service_name FROM opencall, updatedb LEFT JOIN sc_folio ON sc_folio.fk_cmdb_id = opencall.itsm_fk_service WHERE updatedb.callref = opencall.callref AND updatedb.udindex = 0 AND callclass = 'Service Request' AND status != 17 AND appcode = 'ITSM' ",
    "CoreFieldMapping": {
      "summary":"[itsm_title]",
      "description":"Supportworks Service Request Reference: [oldCallRef]\n\n[updatetxt]",
      "customerId":"[cust_id]",
      "customerType":"0",
      "ownerId":"[owner]",
      "teamId":"[suppgroup]",
      "status":"[status]",
      "priorityId":"[priority]",
      "categoryId":"[probcode]",
      "impact":"[itsm_impact_level]",
      "urgency":"[itsm_urgency_level]",
      "serviceId":"[service_name]",
      "site":"[site]"
    },
    "AdditionalFieldMapping":{
      "h_withinfix":"[withinfix]",
      "h_withinresponse":"[withinresp]",
      "h_request_language":"en-GB",
      "h_external_ref_number":"[oldCallRef]",
      "h_closure_category_id":"[fixcode]",
      "h_custom_a":"",
      "h_custom_b":"",
      "h_custom_c":"",
      "h_custom_d":"",
      "h_custom_e":"",
      "h_custom_f":"",
      "h_custom_g":"",
      "h_custom_h":"",
      "h_custom_i":"",
      "h_custom_j":"",
      "h_custom_k":"",
      "h_custom_l":"",
      "h_custom_m":"",
      "h_custom_n":"",
      "h_custom_o":"",
      "h_custom_p":"",
      "h_custom_q":""
    }
  },
  "ConfChangeRequest": {
    "Import":true,
    "CallClass": "Change Request",
    "DefaultTeam":"",
    "DefaultPriority":"",
    "DefaultService":"",
    "SQLStatement":"SELECT opencall.callref, cust_id, itsm_title, owner, suppgroup, status, updatedb.updatetxt, priority, itsm_impact_level, itsm_urgency_level, withinfix, withinresp, bpm_workflow_id, probcode, fixcode, site FROM opencall, updatedb WHERE updatedb.callref = opencall.callref AND updatedb.udindex = 0 AND callclass = 'Change Request' AND status != 17 AND appcode = 'ITSM' ",
    "CoreFieldMapping": {
      "summary":"[itsm_title]",
      "description":"Supportworks Change Request Reference: [oldCallRef]\n\n[updatetxt]",
      "customerId":"[cust_id]",
      "customerType":"0",
      "ownerId":"[owner]",
      "teamId":"[suppgroup]",
      "status":"[status]",
      "priorityId":"[priority]",
      "categoryId":"[probcode]",
      "impact":"[itsm_impact_level]",
      "urgency":"[itsm_urgency_level]",
      "serviceId":"[service_name]",
      "site":"[site]",
      "changeType":""
    },
    "AdditionalFieldMapping":{
      "h_withinfix":"[withinfix]",
      "h_withinresponse":"[withinresp]",
      "h_request_language":"en-GB",
      "h_external_ref_number":"[oldCallRef]",
      "h_closure_category_id":"[fixcode]",
      "h_custom_a":"",
      "h_custom_b":"",
      "h_custom_c":"",
      "h_custom_d":"",
      "h_custom_e":"",
      "h_custom_f":"",
      "h_custom_g":"",
      "h_custom_h":"",
      "h_custom_i":"",
      "h_custom_j":"",
      "h_custom_k":"",
      "h_custom_l":"",
      "h_custom_m":"",
      "h_custom_n":"",
      "h_custom_o":"",
      "h_custom_p":"",
      "h_custom_q":""
    }
  },
  "ConfProblem": {
    "Import":true,
    "CallClass": "Problem",
    "DefaultTeam":"",
    "DefaultPriority":"",
    "DefaultService":"",
    "SQLStatement":"SELECT opencall.callref, cust_id, itsm_title, owner, suppgroup, status, updatedb.updatetxt, priority, itsm_impact_level, itsm_urgency_level, withinfix, withinresp, bpm_workflow_id, probcode, fixcode, site FROM opencall, updatedb WHERE updatedb.callref = opencall.callref AND updatedb.udindex = 0 AND callclass = 'Problem' AND status != 17 AND appcode = 'ITSM' ",
    "CoreFieldMapping": {
      "summary":"[itsm_title]",
      "description":"Supportworks Problem Reference: [oldCallRef]\n\n[updatetxt]",
      "customerId":"[cust_id]",
      "customerType":"0",
      "ownerId":"[owner]",
      "teamId":"[suppgroup]",
      "status":"[status]",
      "priorityId":"[priority]",
      "categoryId":"[probcode]",
      "impact":"[itsm_impact_level]",
      "urgency":"[itsm_urgency_level]",
      "serviceId":"[service_name]",
      "site":"[site]"
    },
    "AdditionalFieldMapping":{
      "h_withinfix":"[withinfix]",
      "h_withinresponse":"[withinresp]",
      "h_request_language":"en-GB",
      "h_external_ref_number":"[oldCallRef]",
      "h_closure_category_id":"[fixcode]",
      "h_custom_a":"",
      "h_custom_b":"",
      "h_custom_c":"",
      "h_custom_d":"",
      "h_custom_e":"",
      "h_custom_f":"",
      "h_custom_g":"",
      "h_custom_h":"",
      "h_custom_i":"",
      "h_custom_j":"",
      "h_custom_k":"",
      "h_custom_l":"",
      "h_custom_m":"",
      "h_custom_n":"",
      "h_custom_o":"",
      "h_custom_p":"",
      "h_custom_q":""
    }
  },
  "ConfKnownError": {
    "Import":true,
    "CallClass": "Known Error",
    "DefaultTeam":"",
    "DefaultPriority":"",
    "DefaultService":"",
    "SQLStatement":"SELECT opencall.callref, cust_id, itsm_title, owner, suppgroup, status, updatedb.updatetxt, priority, itsm_impact_level, itsm_urgency_level, withinfix, withinresp, bpm_workflow_id, probcode, fixcode, site FROM opencall, updatedb WHERE updatedb.callref = opencall.callref AND updatedb.udindex = 0 AND callclass = 'Known Error' AND status != 17 AND appcode = 'ITSM' ",
    "CoreFieldMapping": {
      "summary":"[itsm_title]",
      "description":"Supportworks Known Error Reference: [oldCallRef]\n\n[updatetxt]",
      "customerId":"[cust_id]",
      "customerType":"0",
      "ownerId":"[owner]",
      "teamId":"[suppgroup]",
      "status":"[status]",
      "priorityId":"[priority]",
      "categoryId":"[probcode]",
      "impact":"[itsm_impact_level]",
      "urgency":"[itsm_urgency_level]",
      "serviceId":"",
      "site":"[site]"
    },
    "AdditionalFieldMapping":{
      "h_withinfix":"[withinfix]",
      "h_withinresponse":"[withinresp]",
      "h_request_language":"en-GB",
      "h_external_ref_number":"[oldCallRef]",
      "h_closure_category_id":"[fixcode]",
      "h_custom_a":"",
      "h_custom_b":"",
      "h_custom_c":"",
      "h_custom_d":"",
      "h_custom_e":"",
      "h_custom_f":"",
      "h_custom_g":"",
      "h_custom_h":"",
      "h_custom_i":"",
      "h_custom_j":"",
      "h_custom_k":"",
      "h_custom_l":"",
      "h_custom_m":"",
      "h_custom_n":"",
      "h_custom_o":"",
      "h_custom_p":"",
      "h_custom_q":""
    },
    "PriorityMapping": {
      "Supportworks Priority":"Service Manager Priority"
    },
    "TeamMapping": {
      "Supportworks Group ID":"Service Manager Team Name"
    },
    "CategoryMapping": {
      "Supportworks Profile Code":"Service Manager Profile Code"
    },
    "ResolutionCategoryMapping": {
      "Supportworks Resolution Profile Code":"Service Manager Resolution Profile Code"
    },
    "ServiceMapping": {
      "Supportworks Service Name":"Service Manager Service Name"
    }
}
```

#### HBConfig
Connection information for the Hornbill instance:
* "UserName" - Instance User Name with which the tool will log the new requests
* "Password" - Instance Password for the above User
* "InstanceId" - Instance Id

#### SWServerAddress
The address of the Supportworks Server. If this tool is to be run on the Supportworks Server, then this should be set to localhost.

#### AttachmentRoot
This is the location of the Supportworks Call File Attachment Store.

#### SWSystemDBConf
Contains the connection information for the Supportworks cache database (sw_systemdb).
* "Driver" the driver to use to connect to the sw_systemdb database:
* swsql = Supportworks 7.x SQL (MySQL v4.0.16). Also supports MySQL v3.2.0 to <v5.0
* mysql = MySQL Server v5.0 or above, or MariaDB (Supportworks v8+)
* "UserName" Username for a user that has read access to the SQL database from the location of the tool
* "Password" Password for above User Name

#### SWAppDBConf
Contains the connection information for the Supportworks application database (swdata).
* "Driver" the driver to use to connect to the database that holds the Supportworks application information:
* swsql = Supportworks 7.x SQL (MySQL v4.0.16). Also supports MySQL v3.2.0 to <v5.0
* mysql = MySQL Server v5.0 or above, or MariaDB (Supportworks v8+)
* mssql = Microsoft SQL Server (2005 or above)
* "Server" The address of the SQL server
* "UserName" The username for the SQL database
* "Password" Password for above User Name
* "Port" SQL port (5002 if the data is hosted on the Supportworks server)
* "Encrypt" Boolean value to specify whether the connection between the script and the database should be encrypted. ''NOTE'': There is a bug in SQL Server 2008 and below that causes the connection to fail if the connection is encrypted. Only set this to true if your SQL Server has been patched accordingly.

#### ConfCallClass
Contains request-class specific configuration. This section should be repeated for all Service Manager Call Classes.
* Import - boolean true/false. Specifies whether the current class section should be included in the import.
* CallClass - specifies the Service Manager request class that the current Conf section relates to.
* DefaultTeam - If a request is being imported, and the tool cannot verify its Support Group, then the Support Group from this variable is used to assign the request.
* DefaultPriority - If a request is being imported, and the tool cannot verify its Priority, then the Priority from this variable is used to escalate the request.
* DefaultService - If a request is being imported, and the tool cannot verify its Service from the mapping, then the Service from this variable is used to log the request.
* SQLStatement - The SQL query used to get call (and extended) information from the Supportworks application data.
* CoreFieldMapping - The core fields used by the API calls to raise requests within Service Manager, and how the Supportworks data should be mapped in to these fields.
* - Any value wrapped with [] will be populated with the corresponding response from the SQL Query
* - Any Other Value is treated literally as written example:
* -- "summary":"[itsm_title]", - the value of itsm_title is taken from the SQL output and populated within this field
* -- "description":"Supportworks Incident Reference: [oldCallRef]\n\n[updatetxt]", - the request description would be populated with "Supportworks Incident Reference: ", followed by the Supportworks call reference, 2 new lines then the call description text from the Supportworks call.
* -- "site":"[site]", - When a string is passed to the site field, the script attempts to resolve the given site name against the Site entity, and populates the request with the correct site information. If the site cannot be resolved, the site details are not populated for the request being imported.
* -- NOTE - Change Requests can have a core variable named changeType. Again, this can be populated either by using the SQL output, or a hard-coded value in the JSON config.
* AdditionalFieldMapping - Contains additional columns that can be stored against the new request record. Mapping rules are as above.

#### PriorityMapping
Allows for the mapping of Priorities between Supportworks and Hornbill Service Manager, where the left-side properties list the Priorities from Supportworks, and the right-side values are the corresponding Priorities from Hornbill that should be used when escalating the new requests.

#### TeamMapping
Allows for the mapping of Support Groups/Team between Supportworks and Hornbill Service Manager, where the left-side properties list the Support Group ID's (not the Group Name!) from Supportworks, and the right-side values are the corresponding Team names from Hornbill that should be used when assigning the new requests.

#### CategoryMapping
Allows for the mapping of Problem Profiles/Request Categories between Supportworks and Hornbill Service Manager, where the left-side properties list the Profile Codes (not the descriptions!) from Supportworks, and the right-side values are the corresponding Profile Codes (again, not the descriptions!) from Hornbill that should be used when categorising the new requests.

#### ResolutionCategoryMapping
Allows for the mapping of Resolution Profiles/Resolution Categories between Supportworks and Hornbill Service Manager, where the left-side properties list the Resolution Codes (not the descriptions!) from Supportworks, and the right-side values are the corresponding Resolution Codes (again, not the descriptions!) from Hornbill that should be used when applying Resolution Categories to the newly logged requests.

#### ServiceMapping
Allows for the mapping of Services between Supportworks and Hornbill Service Manager, where the left-side properties list the Service names from Supportworks, and the right-side values are the corresponding Services from Hornbill that should be used when raising the new requests.

# Execute
Command Line Parameters
* file - Defaults to `conf.json` - Name of the Configuration file to load
* dryrun - Defaults to `false` - Set to True and the XMLMC for new request creation will not be called and instead the XML will be dumped to the log file, this is to aid in debugging the initial connection information.
* zone - Defaults to `eur` - Allows you to change the ZONE used for creating the XMLMC EndPoint URL https://{ZONE}api.hornbill.com/{INSTANCE}/

# Testing
If you run the application with the argument dryrun=true then no requests will be logged - the XML used to raise requests will instead be saved in to the log file so you can ensure the data mappings are correct before running the import.

'sw_call_import.exe -dryrun=true'

# Logging
All Logging output is saved in the log directory in the same directory as the executable the file name contains the date and time the import was run 'SW_Call_Import_2015-11-06T14-26-13Z.log'

# Error Codes
* `100` - Unable to create log File
* `101` - Unable to create log folder
* `102` - Unable to Load Configuration File
