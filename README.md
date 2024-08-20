# ai-dock-comfyui-go
A minimalistic Golang client for ai-dock/comfyui

TBD

## Example

```go
const (
  apiURL   = "https://.../.../api"
  apiToken = "..."
)
c := NewClient(apiURL)
c.APIToken = apiToken

resp := must(c.StartWorkflow(ctx, NewStartWorkflowRequest(prompt)))
id := resp.ID

for {
  time.Sleep(5 * time.Second)
  resp := must(c.WorkflowStatus(ctx, id))
  fmt.Println(resp.Status)
  for _, out := range resp.Output {
    fmt.Println(out.LocalPath)
    fmt.Println(out.GCP)
    fmt.Println()
  }
  if resp.Status == StatusSuccess {
    break
  }
}
```
