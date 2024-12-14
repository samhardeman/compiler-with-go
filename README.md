# Minimal Go Compiler

### Build and Run
How to Run with Go:
```
go run compiler.go optimizer.go tac.go mips.go -file input.josh
```

How to Build:
```
go build compiler.go optimizer.go tac.go mips.go
```

How to Run Binary:
```
./compiler -file input.josh
```

*Make sure to pass in the file with the `-file` flag*

### Types
The compiler supports the following types:
- `string`
- `bool`
- `int`
- `float`
- `char`
- `global` - used to allow access from all subscopes

### Initialize
Syntax
```
[type] [name]
[type] [name] = [value]
global [type] [name]
```

### Functions
Syntax
```
func [name]([type] [param],...) [return type, omit if void] {
    [body]
    [return, omit if void] [returned value, omit if void]
}
```

### Logic
Syntax
```
if ([value] [operator] [value]) {
    [body]
} else {
    [body]
}
```

### Arithmetic
Supported Operators
- `+`
- `-`
- `/`
- `*`
- `%`
Syntax
```
[value] [operator] [value]
```

### Loops
For loops are the only type of loop supported.
Syntax
```
for (int i = 0; i < [value]; i = i + [increment]) {
    [body]
}
```