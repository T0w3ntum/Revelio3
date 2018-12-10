# Revelio3

Perform live host discovery and port profiling even quicker! (probably not, but I got lazy and decided why not.) 

## Step one

Live host discovery! 

```
$ revelio3 -f input.txt
[+] 192.168.0.1
[+] 192.168.0.7
[+] 192.168.0.10
[+] 192.168.0.11
[+] 192.168.0.12
[+] 192.168.0.13
[+] 192.168.0.14
[+] 192.168.0.27
[+] 192.168.0.29
[+] 192.168.0.35
[+] 192.168.0.42
2018/12/09 08:45:36 Finished
```

## Step two

Now find the ports. Output from Step one was saved to livehosts.txt

```
$ revelio3 -f livehosts.txt -p -t 5                                                         
[+] Host: 192.168.0.1 - 80,443
[+] Host: 192.168.0.13 - 8080
[+] Host: 192.168.0.27 - 80,443
[+] Host: 192.168.0.42 - 8009,9000
[+] Host: 192.168.0.42 - 8008,8443
[+] Host: 192.168.0.11 - 111
[+] Host: 192.168.0.1 - 30005
[+] Host: 192.168.0.12 - 8080
[+] Host: 192.168.0.42 - 10001
[+] Host: 192.168.0.1 - 5431
[+] Host: 192.168.0.11 - 1234
[+] Host: 192.168.0.11 - 8099
[+] Host: 192.168.0.27 - 57621,61980
[+] Host: 192.168.0.12 - 7011
[+] Host: 192.168.0.7 - 9295,41800
[+] Host: 192.168.0.13 - 7011
[+] Host: 192.168.0.35 - 9295,63963
[+] Host: 192.168.0.11 - 7892,52113,56789,56790,65528
[+] Host: 192.168.0.14 - 33596,50833,51104,60000
2018/12/10 07:05:51 Finished
```
