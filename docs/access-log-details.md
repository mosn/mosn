
#### AccessLog format consists of following three parts, the keys of each part are linked by "%" and blank space:
```$xslt
part1:RequestInfoFormat
part2:RequestHeaderFormat
Part3:ResponseHeaderFormat
```
##### For part1, the request information contains keys(case sensitive):
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
#####so you can choose above keys optionally to define part1 format such as
```$xslt
RequestInfoFormat = "%StartTime% %Protocol% %ResponseCode%"
```

##### For Part2 and Part3, you should define keys according to protocols you choose in format:

###### For Part2,your format begins with 'REQ.', such as:
```
RequestHeaderFormat = "%REQ.part1% %REQ.part2% %REQ.part3%..."
```
###### For part3, your format begins with "Resp." , such as:
```
ResponseHeaderForamt = "%RESP.part1% %RESP.part2% %RESP.part3%..."
```
#### As a whole, the final format looks like:
```
format = "%StartTime% %Protocol% %ResponseCode% %REQ.part1% %REQ.part2% %RESP.part1% %RESP.part2%"
```
we will parse the details and get the content from headers, then log them