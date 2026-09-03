# deployments/

Live Orkestra deployments — operators that run continuously and serve real traffic.

| Deployment | What it is | Live at |
|------------|------------|---------|
| [`public/`](public/README.md) | Six Orkestra runtimes in one cluster, aggregated by a single Control Center. No login — visitors see real operator activity in real time. | [cc.orkestra.sh](https://cc.orkestra.sh) |

---

This is where the website demo lives. When someone opens [cc.orkestra.sh](https://cc.orkestra.sh) from the Orkestra homepage, they are watching these operators reconcile.

New deployments go in their own subdirectory here, following the same structure as `public/`: a `Makefile` for lifecycle operations, one directory per runtime instance, and a `values.yaml` for shared Helm configuration.
