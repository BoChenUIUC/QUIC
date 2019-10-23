#!/bin/bash
for ping_interval in {10..60..10}
do
   for xack_interval in {10..100..10}
   do
     for rep in {1..1}
     do
       echo $ping_interval $xack_interval $rep
       ssh -i "/home/bo/code/quic-go-0.12.0/example/gaming/cloud/gaming.pem" ec2-user@ec2-3-16-41-32.us-east-2.compute.amazonaws.com go run "/home/ec2-user/work/QUIC/example/gaming/main/server.go" 1 5000 $ping_interval $xack_interval $rep
       go run "/home/bo/code/quic-go-0.12.0/example/gaming/main/client.go" 1 5000 $ping_interval $xack_interval $rep
     done
   done
done
