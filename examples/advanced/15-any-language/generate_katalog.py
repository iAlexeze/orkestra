#!/usr/bin/env python3
import yaml
import sys

def generate_katalog(app_name, image, port, replicas=2, ingress_host=""):
    """
    Generates an Orkestra Katalog YAML for a basic web app.
    """
    katalog = {
        "apiVersion": "orkestra.orkspace.io/v1",
        "kind": "Katalog",
        "metadata": {
            "name": f"{app_name}-operator",
            "description": f"Generated Katalog for {app_name}"
        },
        "spec": {
            "crds": {
                f"{app_name}": {
                    "apiTypes": {
                        "group": "web.orkestra.io",
                        "version": "v1alpha1",
                        "kind": "WebApp",
                        "plural": "webapps"
                    },
                    "operatorBox": {
                        "reconciler": {
                            "workers": 2,
                            "resync": "30s",
                        },
                        "onCreate": {
                            "deployments": [
                                {
                                    "name": "{{ .metadata.name }}-deploy",
                                    "image": image,
                                    "replicas": str(replicas),
                                    "port": port,
                                    "reconcile": True
                                }
                            ],
                            "services": [
                                {
                                    "name": "{{ .metadata.name }}-svc",
                                    "port": port,
                                    "targetPort": port,
                                    "reconcile": True
                                }
                            ]
                        }
                    }
                }
            }
        }
    }

    # Conditionally add Ingress if host provided
    if ingress_host:
        ingress = {
            "ingresses": [
                {
                    "name": "{{ .metadata.name }}-ingress",
                    "host": ingress_host,
                    "serviceName": "{{ .metadata.name }}-svc",
                    "servicePort": port,
                    "className": "nginx",
                    "reconcile": True
                }
            ]
        }
        # Add ingress block under operatorBox.onCreate
        katalog["spec"]["crds"][app_name]["operatorBox"]["onCreate"].update(ingress)

    return katalog

if __name__ == "__main__":
    # Example usage – could read from environment or CLI args
    app = "myapp"
    img = "nginx:1.25"
    p = 8080
    replicas = 3
    host = "myapp.example.com"   # set to empty string to skip Ingress

    k = generate_katalog(app, img, p, replicas, host)
    yaml.dump(k, sys.stdout, default_flow_style=False)

