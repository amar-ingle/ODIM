#!/bin/bash
# (C) Copyright [2020] Hewlett Packard Enterprise Development LP
#
# Licensed under the Apache License, Version 2.0 (the "License"); you may
# not use this file except in compliance with the License. You may obtain
# a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
# WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
# License for the specific language governing permissions and limitations
# under the License.

declare PID=0

sigterm_handler()
{
        if [[ $PID -ne 0 ]]; then
                # will wait for other instances to gracefully announce quorum exit
                sleep 5
                kill -9 $PID
                wait "$PID" 2>/dev/null
        fi
        exit 0
}

# create a signal trap
create_signal_trap()
{
        trap 'echo "[$(date)] -- INFO  -- SIGTERM received for systems, initiating shut down"; sigterm_handler' SIGTERM
}

# keep the script running till SIGTERM is received
run_forever()
{
        wait
}

start_systems()
{
	cd /bin
	export CONFIG_FILE_PATH=/etc/odimra_config/odimra_config.json
        registry_address="consul:8500"
        if [ $HA_ENABLED == true ]; then
                registry_address="[consul1:8500,consul2:8500,consul3:8500]"
        fi
	nohup ./svc-systems --registry=consul --registry_address=${registry_address} --server_address=systems:45104 --client_request_timeout=`expr $(cat $CONFIG_FILE_PATH | grep SouthBoundRequestTimeoutInSecs | cut -d : -f2 | cut -d , -f1 | tr -d " ")`s >> /var/log/odimra_logs/systems.log 2>&1 &
	PID=$!
	sleep 2s
	nohup /bin/add-hosts -file /tmp/host.append >> /var/log/odimra_logs/add-hosts.log 2>&1 &
}

monitor_process()
{
        while true; do
                pid=$(pgrep svc-systems 2> /dev/null)
                if [[ $pid -eq 0 ]]; then
                        echo "svc-systems has exited"
                        exit 1
                fi
                sleep 5
        done &
}

##############################################
###############  MAIN  #######################
##############################################

add_host

start_systems

create_signal_trap

monitor_process

run_forever

exit 0
