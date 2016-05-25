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
