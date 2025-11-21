# 🏠

Megarepo for my personal [homelab](https://www.reddit.com/r/homelab/wiki/introduction/)

## Hardware

### Networking

 - [UniFi Dream Machine Special Edition (UDM-SE)](https://ca.store.ui.com/ca/en/category/all-cloud-gateways/products/udm-se)
 - [UniFi 8-Port Aggregation Switch (USW-Aggregation)](https://ca.store.ui.com/ca/en/products/usw-aggregation)
 - [UniFi 24-Port PoE Switch (USW-24-POE)](https://ca.store.ui.com/ca/en/products/usw-24-poe)

### Mini 10" 8U Rack

#### 10" 8U Rack

 - Custom built using 2020 aluminum extrusions and 3D printed components
    - 2x [2020 Carrying Handles](https://makerworld.com/en/models/160002-voron-2-4-trident-carrying-handle), printed with [Fiberon PETG-rCF08](https://ca.polymaker.com/products/fiberon-petg-rcf08)
    - 8x [2020 Corner Brackets](https://makerworld.com/en/models/694515-2020-corner-bracket-m5-version), printed with [Fiberon PETG-rCF08](https://ca.polymaker.com/products/fiberon-petg-rcf08)
    - 4x [2020 Leveler Pads](https://makerworld.com/en/models/1055083-2020-leveler-pad-for-aluminum-extrusions), printed with [Fiberon PETG-rCF08](https://ca.polymaker.com/products/fiberon-petg-rcf08)
    - 1x [1U 2.5"x4 Hot Swap Bays](https://makerworld.com/en/models/1648104-10-inch-rack-1u-4x-2-5-inch-hdd-ssd-hot-swap), printed with [Fiberon PA6-CF20](https://ca.polymaker.com/products/fiberon-pa6-cf20)
    - 3x [1U 3.5"x2 Hot Swap Bays](https://makerworld.com/en/models/1400538-10-inch-rack-1u-2-x-3-5-inch-hdd-hot-swap), printed with [Fiberon PA6-CF20](https://ca.polymaker.com/products/fiberon-pa6-cf20)
    - 1x [1U Power Distribution Unit](makerworld.com/en/models/1691049-10-rack-power-distribution-unit), printed with [Fiberon PA6-CF20](https://ca.polymaker.com/products/fiberon-pa6-cf20)

#### Storage

Current total raw storage capacity is just 79 TB. This will be expanded once the AI / LLM hyper train derails and storage becomes reasonably priced again.

 - 7x 1TB Samsung 990 PRO NVMe SSDs
 - 4x 2TB Crucial MX500 3D NAND SATA SSDs
 - 6x 16TB WD Ultrastar DC HC550 SATA 3.5" HDDs

#### Compute

 - 3x Cluster Server
    - RaspberryPi 5
      - Broadcom BCM2712 ARM CPU
      - 8 GB Memory
      - [52Pi M.2 NVMe and 2.5 GbE HAT](https://52pi.com/products/52pi-w01-u2500-usb-2-5g-ethernet-nvme-for-raspberry-pi-5)
        - 1TB Samsung 990 PRO NVMe SSD
    - RaspberryPi 5
      - Broadcom BCM2712 ARM CPU
      - 8 GB Memory
      - [52Pi M.2 NVMe and 2.5 GbE HAT](https://52pi.com/products/52pi-w01-u2500-usb-2-5g-ethernet-nvme-for-raspberry-pi-5)
        - 1TB Samsung 990 PRO NVMe SSD
    - RaspberryPi 5
      - Broadcom BCM2712 ARM CPU
      - 8 GB Memory
      - [52Pi M.2 NVMe and 2.5 GbE HAT](https://52pi.com/products/52pi-w01-u2500-usb-2-5g-ethernet-nvme-for-raspberry-pi-5)
        - 1TB Samsung 990 PRO NVMe SSD
 - 2x Cluster Workers/Storage
    - [Topton N18 NAS Motherboard](https://www.aliexpress.com/item/1005005347552418.html) &mdash; Bulk 2.5" SSD Storage
      - Intel N150 x86 CPU
      - 32 GB DDR5 Memory
      - 2x 1TB Samsung 990 PRO NVMe SSD
      - 4x 2TB Crucial MX500 3D NAND SATA SSDs
    - [Topton N18 NAS Motherboard](https://www.aliexpress.com/item/1005005347552418.html) &mdash;  Bulk 3.5" HDD Storage
      - Intel N150 x86 CPU
      - 32 GB DDR5 Memory
      - 2x 1TB Samsung 990 PRO NVMe SSD
      - 6x 16TB WD Ultrastar DC HC550 SATA 3.5" HDDs

### Standard 19" 18U Rack (Deprecated)

 - @TODO: backfill 19" rack details

## Networking

### Layout

| CIDR           | Description                 |
|----------------|-----------------------------|
| `10.32.0.0/14` | UniFi Network VLAN          |
| `10.33.0.0/16` | Cluster Pods                |
| `10.34.0.0/16` | Cluster Services            |
| `10.35.0.0/16` | Cluster External Addresses  |

### Static IPs

| CIDR           | Description                 |
|----------------|-----------------------------|
| `10.32.0.1`    | VLAN Gateway                |
| `10.32.10.1`   | Cluster Server Node         |
| `10.32.10.2`   | Cluster Server Node         |
| `10.32.10.3`   | Cluster Server Node         |

---

## Wishlist

Here is a list of self-hosted applications I might add in the future:

 - https://github.com/TomBursch/kitchenowl
 - https://github.com/manyfold3d/manyfold
 - https://github.com/danielbrendel/hortusfox-web
 - https://github.com/Shelf-nu/shelf.nu
