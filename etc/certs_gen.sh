#!/bin/bash

# not in mac os
if [ "$(uname)" = "Linux" ]; then
    echo "Running on Linux"

    mkdir -p /opt/job-worker/certs

    cd /opt/job-worker/certs
else
    echo "For Dev"

    mkdir -p ../certs

    cd ../certs
fi

# CA cert
openssl genrsa -out ca-key.pem 4096

openssl req -new -x509 -days 1095 -key ca-key.pem -out ca-cert.pem -subj "/C=US/ST=State/L=City/O=Ehsaniara/OU=Security/CN=JobWorker-CA"

# Server cert
openssl genrsa -out server-key.pem 2048

openssl req -new -key server-key.pem -out server.csr -subj "/C=US/ST=State/L=City/O=Ehsaniara/OU=Security/CN=job-worker.ehsaniara.com"

# SAN (Subject Alternative Names)
cat > server-ext.cnf << EOF
[req]
req_extensions = v3_req
distinguished_name = req_distinguished_name

[req_distinguished_name]

[v3_req]
basicConstraints = CA:FALSE
keyUsage = digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names

[alt_names]
DNS.1 = job-worker.ehsaniara.com
DNS.2 = job-worker
IP.1 = 192.168.1.161
IP.2 = 0.0.0.0
EOF

openssl x509 -req -days 365 -in server.csr -CA ca-cert.pem -CAkey ca-key.pem -CAcreateserial -out server-cert.pem -extensions v3_req -extfile server-ext.cnf

openssl verify -CAfile ca-cert.pem server-cert.pem

# Admin Client cert
openssl genrsa -out admin-client-key.pem 2048

openssl req -new -key admin-client-key.pem -out admin-client.csr -subj "/C=US/ST=State/L=City/O=Ehsaniara/OU=Admin/CN=job-worker-admin-client"

cat > admin-client-ext.cnf << EOF
[req]
req_extensions = v3_req
distinguished_name = req_distinguished_name

[req_distinguished_name]

[v3_req]
basicConstraints = CA:FALSE
keyUsage = digitalSignature, keyEncipherment
extendedKeyUsage = clientAuth
subjectAltName = @alt_names

[alt_names]
# Using DNS entry to embed role information
DNS.1 = admin.job-worker-client.local
EOF

openssl x509 -req -days 365 -in admin-client.csr -CA ca-cert.pem -CAkey ca-key.pem -CAcreateserial -out admin-client-cert.pem -extensions v3_req -extfile admin-client-ext.cnf

openssl verify -CAfile ca-cert.pem admin-client-cert.pem

# Viewer Client cert
openssl genrsa -out viewer-client-key.pem 2048

openssl req -new -key viewer-client-key.pem -out viewer-client.csr -subj "/C=US/ST=State/L=City/O=Ehsaniara/OU=Viewer/CN=job-worker-viewer-client"

cat > viewer-client-ext.cnf << EOF
[req]
req_extensions = v3_req
distinguished_name = req_distinguished_name

[req_distinguished_name]

[v3_req]
basicConstraints = CA:FALSE
keyUsage = digitalSignature, keyEncipherment
extendedKeyUsage = clientAuth
subjectAltName = @alt_names

[alt_names]
# Using DNS entry to embed role information
DNS.1 = viewer.job-worker-client.local
EOF

openssl x509 -req -days 365 -in viewer-client.csr -CA ca-cert.pem -CAkey ca-key.pem -CAcreateserial -out viewer-client-cert.pem -extensions v3_req -extfile viewer-client-ext.cnf

openssl verify -CAfile ca-cert.pem viewer-client-cert.pem


# server certs
chmod 600 server-key.pem ca-key.pem
chmod 644 server-cert.pem ca-cert.pem

# client certs
chmod 600 admin-client-key.pem viewer-client-key.pem
chmod 644 admin-client-cert.pem viewer-client-cert.pem

rm -f *.csr *.cnf

