def run(r):
  for resource in r:
    resource["metadata"]["labels"]["foo"] = "bar"

run(ctx.resource_list["items"])
