toxiproxy-cli delete redis1_latency
kill -9 $(ps aux | grep [t]oxiproxy | awk '{print $2}')
