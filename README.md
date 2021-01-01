#Ratelimit

Cli tool for restrictions on the launch speed and the number of commands run

For each line of stdin, the utility runs the arguments specified on the command line. 
The commands are run no more often than specified in the "rate" argument. 
The utility is able to run multiple commands in parallel. 
At any given time, no more than specified in the "inflight" command is running.

---

example1
```bash
for i in {1..60} ; do echo $i ; done | go run cmd/ratelimiter/main.go -m=echo -rate=15 -inflight=1 -time=true
```

example2
```bash
(echo 1 ; sleep 3 ; echo 2 ; echo 3) | go run main.go -m=echo -rate=2 -inflight=1
```
