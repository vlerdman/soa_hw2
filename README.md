# SOA_HW2: MAFIA

## Client help

```bash
  usage:
        
    get_state - get current state

    check {username} - check role of player (allowed only for sheriff during night)

    vote {username} - vote for jailing or killing player (jailing allowed only during day, killing - during night for mafia)
```

## Build and run docker

### Build and run server

```bash
docker build -f server.Dockerfile -t vlerdman/soa_hw2_server . && docker run -it --name mafiaserver -p 9000:9000 vlerdman/soa_hw2_server
```

### Build and run client (4 separate clients, bots aren't supported yet)

```bash
docker build -f client.Dockerfile -t vlerdman/soa_hw2_client . && docker run -it --name mafiaclient1 --link mafiaserver:mafiaserver vlerdman/soa_hw2_client
```

```bash
docker build -f client.Dockerfile -t vlerdman/soa_hw2_client . && docker run -it --name mafiaclient2 --link mafiaserver:mafiaserver vlerdman/soa_hw2_client
```

```bash
docker build -f client.Dockerfile -t vlerdman/soa_hw2_client . && docker run -it --name mafiaclient3 --link mafiaserver:mafiaserver vlerdman/soa_hw2_client
```

```bash
docker build -f client.Dockerfile -t vlerdman/soa_hw2_client . && docker run -it --name mafiaclient4 --link mafiaserver:mafiaserver vlerdman/soa_hw2_client
```