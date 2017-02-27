# cabi-testgen

Rudimentary test harness for C-ABI testing. Randomly generates a function
with some interesting signature plus code to call it with specific values.
Client can then compile one side (caller vs callee) with a "good" compiler 
and the other side with a "bad" compiler, then test to see if values
are passed/returned correctly.
