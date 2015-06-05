#!/bin/bash

cfg_key="key.pem"
cfg_cert="cert.pem"

# SUDO
if [ $EUID != 0 ]; then
    sudo "$0" "$@"
    exit $?
fi

echo "SELF SIGNED CERTIFICATE GENERATION"

echo "Checking for openssl..."
# check if openssl is installed
#RES=$(openss3l version)
#echo ${RES:0:7}
type openssl >/dev/null 2>&1
if [[ "$?" != "0" ]]; then
	#statements
	echo "Error: OpenSSL not found!"
	exit
else
	RES=$(openssl version)
	echo "OpenSSL is installed. $RES"
fi

echo "Now openssl will generate a self signed certificate."
echo "Country (e.g. US, BR, CN):"
read -e country
echo "State (e.g. New York):"
read -e state
echo "Domain (e.g. localhost, mydomain.com):"
read -e domain
echo "Key output file (e.g. key.pem):"
read -e ifile
if [[ ${#ifile} > 1 ]]; then
	cfg_key=$ifile
fi
echo "Cert output file (e.g. cert.pem):"
read -e ifile
if [[ ${#ifile} > 1 ]]; then
	cfg_cert=$ifile
fi
openssl req -x509 -nodes -days 365 -subj "/C=$country/ST=$state/CN=$domain" -newkey rsa:1024 -keyout $cfg_key -out $cfg_cert

echo "DONE!"