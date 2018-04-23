
#### Access Log Format:
```$xslt

RequestInfoFormat = "%StartTime% %Protocol% "

```

#### and Request Information Contains:
+ StartTime  
+ RequestReceivedDuration
+ ResponseReceivedDuration
+ BytesSent
+ BytesReceived
+ Protocol
+ ResponseCode
+ Duration
+ ResponseFlag
+ UpstreamLocalAddress
+ DownstreamLocalAddress

#### For Request headers and response headers, you can customized your details according with protocol you choose in format: 

```
"%REQ.part1% %REQ.part2% %REQ.part3%..."
"%RESP.part1% %RESP.part2% %RESP.part3%..."

Take boltv1 for example:
Request Headers: %REQ.RequestID% %REQ.Version% %REQ.CmdType%
Response Headers: %RESP.RequestID% %RESP.ResponseStatus%

```

we will parse the details and get the content from headers, then log them