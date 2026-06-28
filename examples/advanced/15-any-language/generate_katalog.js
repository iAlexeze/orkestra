#!/usr/bin/env node
const fs = require('fs');
const yaml = require('js-yaml');

function generateKatalog(appName, image, port, replicas, ingressHost) {
  const katalog = {
    apiVersion: 'orkestra.orkspace.io/v1',
    kind: 'Katalog',
    metadata: {
      name: `${appName}-operator`,
      description: `Generated Katalog for ${appName}`,
    },
    spec: {
      crds: {
        [appName]: {
          apiTypes: {
            group: 'web.orkestra.io',
            version: 'v1alpha1',
            kind: 'WebApp',
            plural: 'webapps',
          },
          operatorBox: {
            reconciler: {
              workers: 2,
              resync: '30s',
            },
            onCreate: {
              deployments: [
                {
                  name: '{{ .metadata.name }}-deploy',
                  image: image,
                  replicas: String(replicas),
                  port: port,
                  reconcile: true,
                },
              ],
              services: [
                {
                  name: '{{ .metadata.name }}-svc',
                  port: port,
                  targetPort: port,
                  reconcile: true,
                },
              ],
            },
          },
        },
      },
    },
  };

  if (ingressHost) {
    katalog.spec.crds[appName].operatorBox.onCreate.ingresses = [
      {
        name: '{{ .metadata.name }}-ingress',
        host: ingressHost,
        serviceName: '{{ .metadata.name }}-svc',
        servicePort: port,
        ingressClass: 'nginx',
        reconcile: true,
      },
    ];
  }

  return katalog;
}

// Command line argument parsing
const args = process.argv.slice(2);
const options = {
  name: 'myapp',
  image: 'nginx:1.25',
  port: 8080,
  replicas: 2,
  ingress: '',
};

for (let i = 0; i < args.length; i++) {
  switch (args[i]) {
    case '--name': options.name = args[++i]; break;
    case '--image': options.image = args[++i]; break;
    case '--port': options.port = parseInt(args[++i], 10); break;
    case '--replicas': options.replicas = parseInt(args[++i], 10); break;
    case '--ingress': options.ingress = args[++i]; break;
    default: console.error(`Unknown option: ${args[i]}`); process.exit(1);
  }
}

const k = generateKatalog(options.name, options.image, options.port, options.replicas, options.ingress);
console.log(yaml.dump(k, { indent: 2 }));