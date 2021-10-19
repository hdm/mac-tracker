# Historical Tracking of MAC Address Assignments

This repository is used to track allocations and modifications of IEEE allocated hardware address ranges.

## Usage
This dataset is updated daily from the IEEE CSV files and new file revisions are checked into the main branch as updates are found. If you would like to use this dataset to determine the approximate age of a MAC address, the [mac-ages](https://github.com/hdm/mac-ages) repository provides a daily-updated CSV file. 

If you would like to maintain a fork of this repository, you need a system with a recent version of Ruby (2.2+), and to run the `update` script in the main directory at whatever interval makes sense. This script will load the current dataset, download the IEEE CSV files, update records as necessary, save the new dataset, and commit the results back to the repository, pushing changes to the `main` branch of the remote `origin`.

## History

The dataset was bootstrapped using a snapshot from the DeepMAC project and the Wireshark commit archives.
