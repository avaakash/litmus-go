#!/bin/bash
if  [[ "${InstallDependency}" == True ]] ; then
    if [[ "$( which toxiproxy 2>/dev/null )" ]] ; then echo Dependency is already installed. 
    else 
        echo "Installing required dependencies" 
        if cat /etc/issue | grep -i Ubuntu ; then
            sudo apt-get update -y
            wget -O toxiproxy-2.1.4.deb https://github.com/Shopify/toxiproxy/releases/download/v2.1.4/toxiproxy_2.1.4_amd64.deb
            sudo dpkg -i toxiproxy-2.1.4.deb
            sudo service toxiproxy start
        else
            echo "There was a problem installing dependencies."
            exit 1
        fi
    fi
fi

toxiproxy-server > /dev/null 2>&1 &
sleep 2s
toxiproxy-cli create ${ToxicName} --listen localhost:${ListenPort} --${StreamType} localhost:${StreamPort}
toxiproxy-cli toxic add ${ToxicName} --type ${ToxicType} --attribute ${ToxicType}=${ToxicValue}