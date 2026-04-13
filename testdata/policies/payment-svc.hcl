path "secret/data/payment-svc/*" {
  capabilities = ["read", "list"]
}

path "secret/data/billing-svc/*" {
  capabilities = ["deny"]
}

path "pki_int/issue/payment-svc" {
  capabilities = ["create", "update"]
}

path "transit/encrypt/payment-key" {
  capabilities = ["update"]
}

path "transit/decrypt/payment-key" {
  capabilities = ["update"]
}
