# Historical Tracking of MAC Address Assignments

This repository is used to track allocations and modifications of IEEE allocated hardware address ranges. The dataset was bootstrapped
using a combination of the [DeepMAC](http://www.deepmac.org) and [Wireshark](http://www.wireshark.org) archives and maintained via daily pulls from the IEEE website.

## Usage
This dataset is updated daily from the IEEE CSV files and new file revisions are checked into the master branch as updates are found. If you would like to use this dataset to determine the approximate age of a MAC address, the [mac-ages](https://github.com/hdm/mac-ages) repository provides a daily-updated CSV file. 

If you would like to maintain a fork of this repository, you need a system with a recent version of Ruby (2.2+), and to run the `update` script in the main directory at whatever interval makes sense. This script will load the current dataset, download the IEEE CSV files, update records as necessary, save the new dataset, and commit the results back to the repository, pushing changes to the `master` branch of the remote `origin`.
