## Nomad Dispatch Hack
2024-11-12

What if we allowed dispatches to block until a result was available?

```
$ nomad job run example.nomad.hcl
$ time echo '{"Meta": {"echo": "Hello World!"}, "WaitForResult": true}' | nomad operator api /v1/job/example/dispatch | jq -r '.Result.Payload | @base64d'
Hello World!

real    0m0.533s
user    0m0.040s
sys     0m0.019s
```


```hcl
job "example" {
  type = "batch"

  parameterized {
    meta_required = ["echo"]
  }

  group "g" {

    task "python" {
      driver = "docker"

      config {
        image   = "python:alpine"
        command = "python"
        args = [
          "-c",
          "import os; open('/alloc/data/result', 'w').write(os.getenv('NOMAD_META_echo'))"
        ]
        auth_soft_fail = true
      }

      resources {
        cpu    = 500
        memory = 256
      }
    }
  }
}
```
