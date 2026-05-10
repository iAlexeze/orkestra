for i in $(seq 1 200); do
  kubectl apply -f - <<EOF
apiVersion: autoscale.orkestra.io/v1alpha1
kind: Ingestor
metadata:
  name: ingestor-$i
spec:
  image: nginx
  replicas: 1
EOF
done
