hwclock -s
/geth --networkid 1337 --rpc --rpcaddr 0.0.0.0  --ws --wsaddr 0.0.0.0 --wsorigins="*" --rpcapi eth,web3,personal,admin,net --dev &
/bin/sleep 5s
/geth --exec "['0x9cd367d79929c56db39d04179662a193a14f0957','0xc73bd0faf3e9bcd8d619413f35de3769acfbcd3a','0xb58159e7c2efd4c6c40c65f1f01b1a357f6fc479'].forEach(function(a) { console.log(a,eth.sendTransaction({from:eth.coinbase, to:a, value: web3.toWei(100,'ether') }))})" attach http://localhost:8545
/bin/sleep 1d
