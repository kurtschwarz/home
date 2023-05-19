# 🏠

Megarepo for my personal [homelab](https://www.reddit.com/r/homelab/wiki/introduction/)

## Hardware

I'll make a pretty diagram and or include some photos at some point

### Rack

 - 1x [StarTech.com 18U 19in Open Frame Server Rack](https://www.startech.com/en-ca/server-management/4postrack18u)

### Power Management

 - 1x [Square D Home Electronics Protective Device 80 kA](https://www.se.com/ca/en/product/HEPD80C/spd%2C-hepd-type-1%2C-120-240-v%2C-1-ph%2C-3-wire%2C-80-ka%2C-consumer-packaging/?range=61969-square-d-hepd-home-electronics-protective-device&node=12368269215-hepd&selected-node-id=12368269215)
 - 2x [Leviton 20 Amp Hospital Grade Surge Outlet](https://www.leviton.com/en/products/t8380-w)
   - Each outlet has a dedicated 20A breaker, this was done for a future deployment of multiple UPS each would get their own outlet / breaker
 - 1x [APC AP7801 Metered Rack PDU](https://www.apc.com/shop/ca/en/products/Rack-PDU-Metered-1U-20A-120V-8-5-20/P-AP7801)
   - Version 1 of the power managment in the rack, UPS coming soon
 - 1x [StarTech.com 1U 19in Metal Rackmount Cable Management Panel](https://www.startech.com/en-ca/server-management/cablmanager2)
   - This is used to tidy up the power cables coming to the bottom of the rack into the APC AP7801 PDU

### Networking

We've recently upgraded from 1000 Mbps cable internet to 8 Gbps fiber. We're slowing upgrading our networking hardware to 10GbE to get the most out of it.

 - 1x [Ubiquiti Dream Machine Special Edition](https://store.ui.com/collections/unifi-network-unifi-os-consoles/products/dream-machine-se)
 - 1x [Ubiquiti Switch 24 POE](https://store.ui.com/collections/unifi-network-switching/products/usw-24-poe)
 - 2x [Ubiquiti U6 Lite Access Point](https://store.ui.com/collections/unifi-network-wireless/products/u6-lite-us)
 - 1x [Ubiquiti AC Mesh Pro Access Point](https://store.ui.com/collections/unifi-network-wireless/products/unifi-ac-mesh-pro-ap)
 - 1x [TRENDnet 24-Port Keystone Shielded 1U Patch Panel](https://www.trendnet.com/products/patch-panel/24-Port-Blank-Keystone-Shielded-1U-Patch-Panel-TC-KP24S)
 - 1x [TRENDnet 10-Port Gigabit POE+ Switch](https://www.trendnet.com/products/product-detail?prod=190_TPE-1020WS)

### Servers

 - 1x [Dell PowerEdge R240](https://www.dell.com/support/home/en-ca/product-support/product/poweredge-r240/overview)
    - 1x PERC H330
      - 4x 8TB WD Red Plus NAS 3.5" Drive (32TB Total)
 - 1x [Dell PowerEdge R620](https://www.dell.com/support/home/en-ca/product-support/product/poweredge-r620/overview)
    - 2x Intel(R) Xeon(R) CPU E5-2620 @ 2.00GHz
    - 6x 16GB DIMM DDR3 Synchronous Registered (Buffered) 1333 MHz (96GB Total)
    - 1x NVIDIA Quadro M20004GB GDDR5 GPU
    - 1x GLOTRENDS M.2 PCIe NVMe Adapter
      - 1x 250GB Samsung 970 EVO Plus NVMe M.2 Internal SSD (Boot Drive)
    - 1x PERC H710 (Flashed IT Mode)
      - 1x 1GB Crucial MX500 2.5" SSD (Longhorn Disk)
 - 1x [Dell PowerEdge R430](https://www.dell.com/support/home/en-ca/product-support/product/poweredge-r430/overview)
    - 2x Intel(R) Xeon(R) CPU E5-2660 v4 @ 2.00GHz
    - 2x 8GB DIMM DDR4 Synchronous Registered (Buffered) 2133 MHz (16GB Total)
    - 1x GLOTRENDS M.2 PCIe NVMe Adapter
      - 1x 250GB Samsung 970 EVO Plus NVMe M.2 Internal SSD (Boot Drive)
    - 1x PERC H730 Mini (Flashed IT Mode)
      - 1x 1TB Crucial MX500 2.5" SSD (Longhorn Disk)
 - 1x [Dell PowerEdge R330](https://www.dell.com/support/home/en-ca/product-support/product/poweredge-r330/overview)
    - 1x GLOTRENDS M.2 PCIe NVMe Adapter
      - 1x 250GB Samsung 970 EVO Plus NVMe M.2 Internal SSD (Boot Drive)
    - 1x PERC H330 (Flashed IT Mode)
      - 1x 1TB Crucial MX500 2.5" SSD (Longhorn Disk)
      - 6x 2TB Crucial MX500 2.5" SSD (12TB Total)

## Kubernetes

I'm using [k3s](https://k3s.io) as my Kubernetes flavour. It was super easy to get up and running and is much less resource heavy then other options. I had previously tried to use [microk8s](https://microk8s.io) and ended up with a unrecoverable cluster failure.

### Setting up K3s

```
curl -sfL https://get.k3s.io | sh -s - server \
  --cluster-init \
  --cluster-cidr 10.34.0.0/16 \
  --service-cidr 10.35.0.0/16 \
  --cluster-dns 10.35.5.5 \
  --write-kubeconfig-mode 644 \
  --bind-address 0.0.0.0 \
  --advertise-address 10.33.0.1 \
  --disable traefik \
  --disable servicelb \
  --disable local-storage \
  --flannel-backend=host-gw
```

## Networking

### Layout

| CIDR          | Description                 |
|---------------|-----------------------------|
| `10.33.0.0/12` | UniFi VLAN                 |
| `10.34.0.0/16` | K3s Pods                   |
| `10.35.0.0/16` | K3s Services               |
| `10.36.0.0/16` | MetalLB External Addresses |

## Deployment Order

```
just deploy system/metal-lb
just deploy system/twingate
just deploy system/traefik
just deploy system/cert-manager
just deploy system/longhorn
```
