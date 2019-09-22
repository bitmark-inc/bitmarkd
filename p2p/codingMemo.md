# Coding Memo

## A pointer or A value ?
Basic Rules but not a fixed one
### In parameters
+  Avoid modify the value : pass value
+  A small struct: pass value 
+  A large struct : pass pointer to increase speed

### In Return
+ Avoid modify the value : return value
+  [Slices/ maps / channels/ strings /function /interface ] : return value because they are implemented with pointers internally, and a pointer to them is often redundant
+ Large Struct : return value

## Performance Advice
+ use  value , not pointer
    ```var Objects [100]Object``
+ use  specific intger/float/...
    ``` a uint32  ```
+  Function calls have an unavoidable overhead. use wisely
+ Escape analysis 