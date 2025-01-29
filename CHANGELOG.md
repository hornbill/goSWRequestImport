# CHANGELOG

## 1.22.1 (January 29th, 2025)

### Feature/Fix

- Fixed database connection issue for connecting to MariaDB System/Cache DB on CS > 6.
- Fixed issue with Historic Updates having extraneous escaping

## 1.22.0 (July 28th, 2023)

### Feature/Fix

- Fixed database connection issue for connecting to System/Cache DB on CS > 6.

## 1.21.0 (February 22nd 2023)

Change:

- Compiled with latest Go binaries because of security advisory.

## 1.15.0 (January 31st, 2023)

### Feature/Fix

- Replacement of .swm translation code to allow for import of Sw email files.

## 1.14.0 (May 20th, 2022)

### Feature 

- Added logic to create initial status history record, to support changes in Service Manager

## 1.13.2 (April 1st, 2022)

Change:

- Updated code to support application segregation

## 1.13.1 (January 28th, 2022)

Fix:
- Fixed prefix issue caused by API change

## 1.13.0 (November 22nd, 2021)

Changes:

- Addition of ODBC as database driver
-- lowercase "odbc" allows for DSN/User/Password
-- UPPERCASE "ODBC" allows for ConnectionString to be used in conf.json
- Modifications to logging.
-- new command line flag (-splitlogs) to split out logs in three.
- Amount of calls returned from Sw now in totals tally.

## 1.12.1 (July 6th, 2021)

Change:

- Rebuilt using latest version of goApiLib, to fix possible issue with connections via a proxy

## 1.12.0 (June 23rd, 2021)

Changes:

- Removed references to content location for file attachments, as this is no longer required
- Added support for Release Request workflows to be spawned 

##�1.11.0 (August 25th, 2020)

Changes:

- Set h_archived column to 1 when requests are being imported in a cancelled state
- Changed BPM spawning to use processSpawn2 instead of processSpawn
- Added version checking code

## 1.10.2 (August 4th, 2020)

Fix:

- Fixed issue with owner mapping

##�1.10.1 (April 15th, 2020)

Change:

- Updated code to support Core application and platform changes

## 1.10.0 (January 13th, 2020)

Changes:

- Can now define a custom query to retrive historic update records;
- Improved nil value handling when writing historic update records;

## 1.9.0 (December 19th, 2019)

Changes:

- Extended logging when -debug=true CLI flag is set
- Fixed intermittent issue with the updating of the log date
- Fixed intermittent issue with the updating of calls to an on-hold status

## 1.8.1 (November 21st, 2019)

Defect Fix:

- Issue resolving sites from mapped site name

## 1.8.0 (November 19th, 2019)

Changes:

- Updated Site search to use h_site mapping value instead of h_site_id
- Changed CLI parameter name from contactOrg to custorg, to reflect additional features below when using this flag
- With the above flag being set to true, if the customer being imported is of type User, the tool will now use the users Home Organisation as the Company on logged requests
- Skip mapping of h_org_id, h_company_id and h_company_name if the above flag is set, so we don't overwrite the discovered values with mapped values

Defect Fix:

- When the customer being imported is of type Contact, the tool was not always using the contacts primary organisation as the Organisation on logged requests

## 1.7.1 (November 15th, 2019)

Defect fix:

- Customer association messages were reported as warnings instead of debugs in the log

## 1.7.0 (November 14th, 2019)

Feature:

- The tool now performs file attachment processing once the requests have been imported

## 1.6.1 (November 4th, 2019)

Defect fix:

- Weave-lab version of MySQL320 has not been available for a while - using the hornbill backup/fork.

## 1.6.0 (November 1st, 2019)

Defect fix:

- Fixed Customer Search

## 1.5.2 (February 18th, 2019)

Defect fix:

- Fixed connection string errors seen when importing historic update records from MSSQL data source

## 1.5.1 (October 23rd 2018)

Defect fix:

- Fixed issue with missing log entries in dryrun mode

## 1.5.0 (October 14th 2018)

Features:

- Performance improvements
- Memory usage improvements
- Reduction in the number of HTTP sessions used
- No longer required to provide instance zone when running tool
- Logging improvements
- Historic Updates and File Attachments are now processed for each request rather than at the end, improving log readability

Defect fix:

- Fixed memory leak

## 1.4.7 (February 7th 2018)

Defect fix:

- Fixed issue with call diary entries sometimes assigning to incorrect request

## 1.4.6 (February 6th 2018)

Features:

- Further improved memory usage

## 1.4.5 (February 2nd 2018)

Features:

- Improved multithreading peformance and memory usage

## 1.4.4 (December 12th 2017)

Defect fix:

- Fixed issue where team mapping could return other groups of the same name.

## 1.4.3 (December 8th 2017)

Features:

- Now removes any incompatible (non UTF-8) characters from the decoded Supportworks Mail file response
- Improved output of SWM file text representation, included RFC headers and made line feeds Windows compatible.

Defect fixes:

- Fixed "null" value issue for date/time of file attachments
- Fixed warnings thrown when date/time is string instead of EPOCH

## 1.4.2 (December 7th 2017)

Defect Fix:

- Corrected issue with historic update attachments being associated to the incorrect historic update within the same request

Feature:

- Improved output when unable to connect to the Hornbill instance

## 1.4.1 (November 30th 2017)

Defect Fix:

- Now sending UTC time to Hornbill APIs to ensure correct log date being applied

## 1.4.0 (November 28th 2017)

Defect Fix:

- Fixed mismatch of historic attachment UpdateID to Historic Update Index

Features:

- Refactored code, broken the code up in the original release in to modules for easier reading and maintenance;
- Status Mapping, which was hard-coded, is now configurable;
- Output DB connection errors to CLI as well as Log
- Optimise hornbillitsmhistoric index once historic updates have been added
- Made the Request Type in the configuration an array of objects, so can add configuration for as many request type imports as necessary.
- Added support for import of Releases;
- Added support for import of BP Task;
- Added support for admin-defined query to return request associations, and configuration to allow imported BP Tasks to be associaed to other imported request types;
- Process file attachments in-line when importing requests, rather than at the end of the import process;
- Supportworks Mail file extraction, and attach files contained in SWM emails as files against new request or historic updates
- Improved logging

## 1.3.0 (January 3rd 2017)

Feature:

- Added logic to write AdditionalFieldMapping mapped data to new RequestsExtended entity

## 1.2.10 (November 25th 2016)

Defect Fix:

- Issue importing requests in an On Hold status in to Service Manager v2.30+

## 1.2.9 (November 14th 2016)

Defect Fix:

- Issue importing file attachments in to historical attachments entity

## 1.2.8 (November 7th 2016)

Feature:

- Historical diary entries are now ordered by descending update time

## 1.2.7 (July 7th, 2016)

Defect Fix:

- File attachments with names containing API-constrained characters [<>|\/:*?"] were not imported. These characters are now replaced by an underscore character _

## 1.2.6 (June 17th, 2016)

Feature:

- Added -attachments flag to allow for the importing of attachments without prompt

Defect Fix:

- Certain imported requests did not have an active activity stream, meaning they could not be updated.

## 1.2.5 (May 31st, 2016)

Defect Fixes:

- Writing of duplicate Request profile codes failing
- Historic Update File Attachments with a .SWM extension importing but not accessible through Service Manager Request UI

## 1.2.4 (May 25th, 2016)

Features:

- Release binary now includes 32bit and 64bit Windows executables
- Added -concurrent flag, allowing you to specify the maximum number of requests to be imported concurrently
- Improved performance when importing file attachments
- Added ability to specify Service Manager Profile Code separation character
- Improved client-side import logging.

## 1.2.3 (May 9th, 2016)

Defect Fixes:

- Fixed data mapping issues
- Fixed issue where imported calls in a status of On Hold could not be taken off hold  

## 1.2.2 (April 29th, 2016)

Features:

- Allow the back-dating of imported requests,to the date/time the original Supportworks request was logged
- Allow the import of Resolved Date & Closed Date to match those of the original Supportworks request
- New requests logged from requests that are On Hold in Supportworks are now placed On Hold in Service Manager, to the original requests date & time

Defect Fixes:

- Fixed issue when importing historical diary entries that have a Time Spent value of NULL

## 1.2.1 (April 28th, 2016)

Features:

- Takes request prefix from Application Settings rather than import tool

## 1.2.0 (April 27th, 2016)

Features:

- Improved import performance:
  - Streamlined API's and client side record caching
  - Multi-threaded the Request Association and File Attachment Import code
- Extended field mapping, allowing more request fields to be written to, including class-specific extended table fields

Defect Fixes:

- Fixed race conditions in Goroutines
- Fixed issue with MSSQL driver returning INT64 values, causing data conversion problems

## 1.1.2 (April 7th, 2016)

Defect Fix:

- Corrected output of oldCallRef mapping variable

## 1.1.1 (April 7th, 2016)

Features:

- Added additional stage in file attachment import user confirmation;
- Enhanced notification display using color in CLI output

## 1.1.0 (April 6th, 2016)

Features:

- Improved import performance using Goroutines & parallel processing of request logging

## 1.0.2 (February 24th, 2016)

Defect Fixes:

- No Default Service being assigned to imported requests when Service Name from Supportworks data contained a NULL value
- Updated Request Status Matrix.

## 1.0.1 (February 1st, 2016)

Features:

- Added missing brace to ConfKnownError section of configuration file.

## 1.0.0 (January 22, 2016)

Features:

- Initial Release
