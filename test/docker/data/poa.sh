NODE1=enode://65c7a27a9d6a289fa206b6a2c47311cac124a89c5d0cd17b9a1d6ef64c1b0aa99e3584ef190fbee6307844e1d5bc9ea316018a147ddbb394a971c1e084fefac4@172.19.0.5:30303
NODE2=enode://3ad8e1fc1fc55b7589235a4f0ffd6a4f04bf5d94339212c32f293d2612233320cb3a3f7931c32aaa63a7d3736eab8f61155ffd6e965b07910b2042e6bc02de38@172.19.0.4:30303
NODE3=enode://d5680da5abf33b66c29960ddb74ba721b18cea3895646d8048aa31181b3ca8574f9c8ea5b50610954abbe4bd59062036f93a42c77566e5128f5c72a24e5bd6f6@172.19.0.3:30303
BOOTNODES=$NODE1,$NODE3,$NODE3 

CFG=/data/$1

hwclock -s
rm -rf /dyndata
mkdir /dyndata
cp -R /data/$1/* /dyndata
/geth init /data/gomet-genesis.json
acc=$(cat  $CFG/account)

/geth --bootnodes $BOOTNODES --networkid 38897 --keystore $CFG/keystore --ws --wsaddr 0.0.0.0 --wsorigins="*" --rpc --rpcaddr 0.0.0.0 --rpcapi eth,web3,personal,clique,admin,net --mine --unlock $acc --password $CFG/password
#echo waiting 10 seconds geth become available
#sleep 10
#/build/gometh-server-linux-amd64 --config /dyndata/gometh.yaml start
