# Allow all read and write permissions
acl {
  policy = "write"
  # Specify the token ID or name that this rule applies to
  # In this case, we use the built-in anonymous token
  token = "anonymous"
  # Specify the resource that this rule applies to
  # In this case, we allow all resources by using the wildcard "*"
  # You can replace this with a specific resource or path as needed
  rules = """
    key "" {
      policy = "write"
    }
    node "" {
      policy = "write"
    }
    service "" {
      policy = "write"
    }
    session "" {
      policy = "write"
    }
  """
}