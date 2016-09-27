# JTP
A broadcast transmission protocol based on UDP.
Go
---

```go
func SendData(){
    s := JTP.NewSender(":45678")
    s.Send([]byte{0,1,2,3,4,5})
    <-time.After(time.Hour)
}
```
```go
func RecvData(){
    r := JTP.NewReceiver()
    r.Dial("127.0.0.1:45678")
    allData := []byte{}
	for {
		data := make([]byte, 1024)
		n, err := r.Read(data)
		allData = append(allData, data[:n]...)
		if err != nil {
			break
		}
	}
	print(allData)
}
```
C#
---
```csharp
void RecvData(){
    var r = new Receiver();
    r.Dial("127.0.0.1", 45678);
    var data = r.Read();
}
```
