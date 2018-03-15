hwclock -s
mkdir /dyndata
cp -fR /data/$1/* /dyndata
/geth --datadir /dyndata init /data/gomet-genesis.json
acc=$(cat /dyndata/account)
/geth --networkid 38897 --datadir /dyndata --ws --wsaddr 0.0.0.0 --wsorigins="*" --rpc --rpcaddr 0.0.0.0 --rpcapi eth,web3,personal,clique,admin,net --mine --unlock $acc --password /dyndata/password
#echo waiting 10 seconds geth become available
#sleep 10
#/build/gometh-server-linux-amd64 --config /dyndata/gometh.yaml start
