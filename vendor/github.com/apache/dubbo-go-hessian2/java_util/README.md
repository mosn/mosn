# UUID-info

1. explain dubbo-go-hessian2 strut UUID
- JavaServer -> create UUID -> GO Client -> UUID struct (PASS)
- dubbo-go-hessian2 cannot create UUID strut
- see jdk source code of class:[java.util.UUID] learning how to create UUID struct
- see https://github.com/satori/go.uuid
2. explain dubbo-go-hession2 strut locale 
-java object locale -> go struct Locale (PASS), but currently implemented are objects enumerated in java. See class:java.util.Locale
-First convert to struct LocaleHandle and then call `GetLocaleFromHandler(localeHandler *LocaleHandle)` function to convert to struct Locale 
-You can use the `language.ParseBase("zh-CN")` function in the `golang.org/x/text/language` package to convert the value of `locale.String()` get go struct and do other things  