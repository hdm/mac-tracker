# Historical Tracking of MAC Address Assignments

This repository is used to track allocations and modifications of IEEE allocated hardware address ranges (MACs).

## Usage

This dataset is updated daily from the IEEE CSV files and new file revisions are checked into the main branch as updates are found.

This repository generates two data files:

* [MAC Tracker JSON](https://raw.githubusercontent.com/hdm/mac-tracker/refs/heads/main/data/macs.json): This is a full JSON dump of each prefix and  going back to around 1998. If you are building a tool to handle MAC lookups, this is the file to work with.

* [MAC Ages CSV](https://raw.githubusercontent.com/hdm/mac-tracker/refs/heads/main/data/mac-ages.csv): This is a simplified CSV that maps each prefix to the earliest registration record. If you need to estimate the age of a device, the initial registration date is a great choice, especially for newer prefixes. 

If you would like to maintain a fork of this repository, you need a system with Go 1.24+ (or Ruby 2.2+ for the legacy version). Build the update tool with `go build -o update update.go` and run the `./update` script in the main directory at whatever interval makes sense. This script will load the current dataset, download the IEEE CSV files, update records as necessary, and save the new dataset. The update script includes automatic retry logic to handle temporary IEEE website outages.

Previously the mac-ages.csv file was updated via a separate repository called `mac-ages`. This secondary repository was archived on June 22, 2025.

## Data Format

The JSON dump is a mapping of prefixes by mask to an array of registration entries. 
Each entry starts with an `add` record and is followed by zero or more `change` records.
Each entry includes the date (`d`), type (`t`), physical address (`a`), associated country (`c`), the organization name (`o`), and the source (`s`) of the records.

In the example below, the prefix `000e02000000` maps the MAC address range `00:0e:02:00:00:00` with a 24-bit (3-byte) mask.


```json
  "000e02000000/24": [
    {
      "d": "2003-09-08",
      "t": "add",
      "a": "657 Orly Ave.\nDorval Quebec H9P 1G1\n\n",
      "c": "CANADA",
      "o": "Advantech AMT Inc.",
      "s": "wireshark.org"
    },
    {
      "d": "2015-08-27",
      "t": "change",
      "a": "657 Orly Ave. Dorval Quebec CA H9P 1G1",
      "c": "CA",
      "o": "Advantech AMT Inc."
    }
  ],
  ```

This 24-bit mask conveniently maps to 3 bytes, and matches all addresses in the form of  `00:0e:02:XX:XX:XX`. 
The mask can vary based on the block size and may be listed as `/24`, `/28`, or `/36`.  The mask refers to the number of leading bits in 
the prefix that match that registration.  A `/36` prefix would mask the first 36 bits of the full 48-bit address, leaving just 12 bits 
for unique addresses, which allows for only 4096 unique MACs. By contrast, the larger `/24` masks allow for 16.7m million unique MACs.

For example, the prefix `70:b3:d5` matches over 4000 separate `/36` prefixes, but also has a `/24` prefix assigned to IEEE.
The IEEE /24 prefix registration points back to the IEEE 36-bit registry.

```json
"70b3d5000000/24": [
    {
      "d": "2014-01-09",
      "t": "add",
      "a": "\n445 HOES LANE\nPISCATAWAY NJ 08854\n",
      "c": "UNITED STATES",
      "o": "IEEE REGISTRATION AUTHORITY  - Please see OUI36 public listing for more information."
    },
```

In the case of MAC address `70:b3:d5:c3:c0:01`, this would match the `/36` prefix `70b3d5c3c000/36`

Practically this means that the larger masks must be matched first to determine the correct registration.
The `mac-ages.csv` is already sorted so that larger prefixes appear first. If you are working with the raw
JSON, you will need to check the `/36` prefixes, then the `/28`, and the finally the `/24` to resolve an
address to an entry.


## Updates

We use GitHub Actions to update this repository twice a day from these IEEE URLs:
 * https://standards-oui.ieee.org/oui/oui.csv
 * https://standards-oui.ieee.org/cid/cid.csv
 * https://standards-oui.ieee.org/iab/iab.csv
 * https://standards-oui.ieee.org/oui28/mam.csv
 * https://standards-oui.ieee.org/oui36/oui36.csv

The GitHub Actions also modify https://raw.githubusercontent.com/hdm/mac-tracker/refs/heads/main/data/updated.txt to include the time of the last sync.

## History

IEEE does not provide historical data feeds and this project was bootstrapped using a snapshot from the DeepMAC project and the Wireshark (previously, Ethereal) commit archives.
The Wireshark project updated their copy of the OUI data roughly once a week.
