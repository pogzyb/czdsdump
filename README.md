# czdsdump

Utility for dumping zone files from the CZDS to an S3 Bucket or FileSystem.

> The Centralized Zone Data Service (CZDS) is an online portal where any interested party can request access to the Zone Files provided by participating generic Top-Level Domains (gTLDs).

Create ICANN account: https://czds.icann.org/home

### Examples

Local
```
git clone https://github.com/pogzyb/czdsdump.git

cd czdsdump
go build -o czdsdump .

./czdsdump all -v -o /tmp -u <user> -p <password>
```

Docker
```
docker pull ghcr.io/pogzyb/czdsdump:latest
docker run -v ./data:/tmp czdsdump all -v -o /tmp -u <user> -p <password>
```

Dump to an S3 bucket
```
# assumes you have aws credentials set in `.env.aws`
docker pull ghcr.io/pogzyb/czdsdump:latest
docker run --env-file .env.aws czdsdump all -v -o s3://mybucket/czds/2024-04-28/ -u <user> -p <password>
```

### Resources / Information

What are these files?

> The registry operator’s zone data contains the mapping of domain names, associated name server names, and IP addresses for those name servers. These details are updated by the registry operator for its respective TLDs whenever information changes or a domain name is added or removed. 
> https://czds.icann.org/help

In short, these files are .txt files containing the domain names for the given registry. For example, the zone file for ".com" would contain all the registered .com domain names at that given time.

How often should you dump these files?

>  ICANN begins the daily collection of zone files from the registry operators at 00:00 UTC, and the process takes no more than 6 hours. This means that all updated zone files are available for download from CZDS after 06:00 UTC. End users of CZDS can freely download each of the latest available zone files once their access request has been approved by the registry operator of the TLD.

> Zone files are updated once per day starting at 00:00 UTC, so an end user of CZDS should only download each TLD zone file a maximum of once per 24-hour period. 
> https://czds.icann.org/help