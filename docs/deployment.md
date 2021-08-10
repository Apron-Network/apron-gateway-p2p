# Deployment

## Download docker image

Please download the latest date of image from [here](https://drive.google.com/drive/folders/1W9X3BAYs9mU2VuBsnPd2axxRtPkXS9co?usp=sharing)


## Load docker image

```
docker load < apron-node-2021xxxx.tar.gz
```

## Dev mode


Please download the dev configuration file from [here](https://github.com/Apron-Network/apron-gateway-p2p/blob/master/full/docker-compose-dev.yml)

```shell 
$ ln -s docker-compose-dev.yml docker-compose.yml
$ docker-compose up -d
```

## A Private Apron Network with Boot and Client Node

### Boot Node Starts firstly

Please download the boot configuration file from [here](https://github.com/Apron-Network/apron-gateway-p2p/blob/master/full/docker-compose-boot.yml)

```shell
$ docker-compose up -d
```
When the node starts you should see output simillar to this using `docker-compose logs`

```


```

### The blockchain expoler 

In your web browser, navigate to http://<boot_addr>:3001/?rpc=ws%3A%2F%2F8.210.85.13%3A9944#/explorer

### Bob Joins

Please download the configuration file from [here](https://github.com/Apron-Network/apron-gateway-p2p/blob/master/full/docker-compose-client.yml)

