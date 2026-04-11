# Permit reading only "secret/foo". an attached token cannot read "secret/food"
# or "secret/foo/bar".
path "secret/foo" {
  capabilities = ["read"]
}

# Permit reading everything under "secret/bar". an attached token could read
# "secret/bar/zip", "secret/bar/zip/zap", but not "secret/bars/zip".
path "secret/bar/*" {
  capabilities = ["read"]
}

# Permit reading everything prefixed with "zip-". an attached token could read
# "secret/zip-zap" or "secret/zip-zap/zong", but not "secret/zip/zap
path "secret/zip-*" {
  capabilities = ["read"]
}
