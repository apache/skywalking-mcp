# MQE (Metrics Query Expression) Detailed Syntax Rules

## 1. Syntax Structure Overview

The basic structure of MQE expressions follows these BNF grammar rules:

```bnf
expression ::= metric_expression | calculation_expression | comparison_expression | function_expression

metric_expression ::= metric_name | metric_name '{' label_selector '}'

label_selector ::= label '=' '"' value_list '"' (',' label '=' '"' value_list '"')*

value_list ::= value (',' value)*

calculation_expression ::= expression operator expression | '(' expression ')'

operator ::= '+' | '-' | '*' | '/' | '%'

comparison_expression ::= expression comparison_operator expression

comparison_operator ::= '>' | '>=' | '<' | '<=' | '==' | '!='

logical_expression ::= expression logical_operator expression

logical_operator ::= '&&' | '||'

function_expression ::= function_name '(' argument_list ')'

argument_list ::= expression (',' expression)*
```

## 2. Core Components Details

### 2.1 Metric Names
- **Rules**: Must be valid metric names, typically lowercase letters separated by underscores
- **Pattern**: `[a-z][a-z0-9_]*`
- **Examples**: `service_sla`, `service_cpm`, `endpoint_resp_time`

### 2.2 Label Selectors
- **Syntax**: `{label='value1,value2,...'}`
- **Multiple labels**: `{label1='value1', label2='value2'}`
- **Special case**: Percentile labels `{p='50,75,90,95,99'}`

### 2.3 Operator Precedence (highest to lowest)
1. Parentheses `()`
2. Function calls
3. Multiplication, division, modulo `*`, `/`, `%`
4. Addition, subtraction `+`, `-`
5. Comparison operators `>`, `>=`, `<`, `<=`, `==`, `!=`
6. Logical AND `&&`
7. Logical OR `||`

## 3. Function Categories and Syntax

### 3.1 Aggregation Functions
```
avg(expression) -> single_value
sum(expression) -> single_value
max(expression) -> single_value
min(expression) -> single_value
count(expression) -> single_value
latest(expression) -> single_value
```

### 3.2 Mathematical Functions
```
abs(expression) -> same_type_as_input
ceil(expression) -> same_type_as_input
floor(expression) -> same_type_as_input
round(expression, decimal_places) -> same_type_as_input
```

### 3.3 Sorting and Selection Functions
```
top_n(metric_name, top_number, order, attrs) -> sorted_list/record_list
  - top_number: positive integer
  - order: 'asc' | 'des'
  - attrs: optional, e.g., attr0='value', attr1='value'

top_n_of(top_n_expr1, top_n_expr2, ..., top_number, order) -> merged_top_n
  - top_n_expr: top_n expressions
  - top_number: positive integer
  - order: 'asc' | 'des'

sort_values(expression, limit, order) -> follows_input_type
  - limit: optional positive integer
  - order: 'asc' | 'des'

sort_label_values(expression, order, label_names...) -> follows_input_type
  - order: 'asc' | 'des'
  - label_names: at least one label name
```

### 3.4 Trend Analysis Functions
```
increase(expression, time_range) -> time_series_values
rate(expression, time_range) -> time_series_values
  - time_range: positive integer, unit aligns with query Step
```

### 3.5 Label Operations
```
relabel(expression, target_label='origin_values', new_label='new_values') -> follows_input
aggregate_labels(expression, aggregation_method(label_names...)) -> time_series_values
  - aggregation_method: sum | avg | max | min
  - label_names: optional, if not specified, all labels will be aggregated
```

### 3.6 Logical Functions
```
view_as_seq([expression1, expression2, ...]) -> follows_selected_expression
is_present([expression1, expression2, ...]) -> single_value (0 or 1)
```

### 3.7 Baseline Functions
```
baseline(expression, baseline_type) -> time_series_values
  - baseline_type: 'value' | 'upper' | 'lower'
```

## 4. Expression Types and Return Values

### 4.1 Return Value Types
- **SINGLE_VALUE**: Single value, like `avg(service_cpm)`
- **TIME_SERIES_VALUES**: Time series data with timestamps
- **SORTED_LIST**: Sorted list of values, like `top_n()`
- **RECORD_LIST**: List of records
- **LABELED_VALUE**: Values with labels, such as percentiles

### 4.2 Type Conversion Rules
- Numeric values can be used directly in arithmetic operations
- Boolean values (comparison results) convert to 0 or 1
- Label values must be accessed through label selectors

## 5. Entity Filtering Rules

### 5.1 Service-Level Filtering
```
expression + entity{serviceName='name', layer='GENERAL', normal=true}
```

### 5.2 Instance-Level Filtering
```
expression + entity{serviceName='name', serviceInstanceName='instance'}
```

### 5.3 Endpoint-Level Filtering
```
expression + entity{serviceName='name', endpointName='endpoint'}
```

### 5.4 Relation Query Filtering
```
expression + entity{
  serviceName='source',
  destServiceName='destination',
  layer='GENERAL',
  destLayer='DATABASE'
}
```

## 6. Common Patterns and Best Practices

### 6.1 Percentage Conversion
```
# Convert SLA from decimal to percentage
service_sla * 100

# Convert CPM to RPS
service_cpm / 60
```

### 6.2 Condition Combinations
```
# High latency with low traffic
service_resp_time > 3000 && service_cpm < 100

# Multiple percentiles exceed threshold
sum(service_percentile{p='50,75,90,95,99'} > 1000) >= 3
```

### 6.3 Trend Monitoring
```
# Response time growth rate
rate(service_resp_time, 5)

# Increase over the past 2 minutes
increase(service_cpm, 2)
```

### 6.4 Aggregation Calculations
```
# Average response time (convert milliseconds to seconds)
avg(service_resp_time) / 1000

# Error rate statistics
sum(aggregate_labels(meter_status_code{status='4xx,5xx'}, sum))
```

## 7. Syntax Validation Rules

### 7.1 Required Conditions
1. Metric names must exist in the system
2. Label names must match the metric type
3. Function parameter count must be correct
4. Types on both sides of operators must be compatible

### 7.2 Common Error Patterns
```
# Error: Missing label selector
service_percentile  # Should be service_percentile{p='50'}

# Error: Invalid aggregation
avg(service_percentile{p='50,75,90'})  # Cannot average multiple values directly

# Error: Type mismatch
"string" + 123  # Cannot add string and number
```

## 8. Parsing Precedence Example

```
# Expression: avg(service_cpm) * 60 > 1000 && service_sla < 0.95
# Parsing order:
1. avg(service_cpm)  # Function call
2. ... * 60          # Multiplication
3. ... > 1000        # Comparison
4. service_sla < 0.95 # Comparison
5. ... && ...        # Logical AND
```

## 9. Advanced Usage

### 9.1 Nested Functions
```
round(avg(service_resp_time) / 1000, 2)  # Average response time in seconds, rounded to 2 decimal places
```

### 9.2 Conditional Aggregation
```
sum((service_sla * 100) < 95)  # Count instances where SLA is below 95%
```

### 9.3 Dynamic Labels
```
relabel(
  service_percentile{p='50,75,90,95,99'}, 
  p='50,75,90,95,99', 
  percentile='P50,P75,P90,P95,P99'
)
```

## 10. Natural Language to MQE Mapping Rules

### 10.1 Keyword Mappings
- "average" → `avg()`
- "maximum" → `max()`
- "minimum" → `min()`
- "total", "sum" → `sum()`
- "count" → `count()`
- "latest" → `latest()`
- "top N" → `top_n(..., N, des)`
- "bottom N" → `top_n(..., N, asc)`
- "percentage" → `* 100`
- "per second" → `/ 60` (for CPM)
- "increase" → `increase()`
- "rate" → `rate()`

### 10.2 Condition Mappings
- "greater than", "more than" → `>`
- "less than", "below" → `<`
- "equals", "is" → `==`
- "not equal", "is not" → `!=`
- "and" → `&&`
- "or" → `||`

### 10.3 Time Range Mappings
- "last hour", "past hour" (past) → `duration: "-1h"`
- "last 30 minutes" (past) → `duration: "-30m"`
- "next 30 minutes" (future) → `duration: "30m"`

## 11. Model Understanding Guidelines

When processing natural language queries from users, the model should:

1. **Identify Intent**: Is it querying a single value, time series, sorted list, or comparison?
2. **Extract Entities**: Service names, instance names, endpoint names, etc.
3. **Identify Metrics**: Response time, success rate, call volume, etc.
4. **Determine Operations**: Aggregation, calculation, comparison, sorting, etc.
5. **Build Expression**: Combine components according to syntax rules

### Example Conversion Process
User input: "Show the average response time for service A in the last hour, alert if it exceeds 1 second"

Analysis steps:
1. Entity: Service A → `service_name: "A"`
2. Metric: Response time → `service_resp_time`
3. Operation: Average → `avg()`
4. Time: Last hour → `duration: "1h"`
5. Condition: Exceeds 1 second → `> 1000` (convert to milliseconds)

Final expression:
```
expression: "avg(service_resp_time) > 1000"
service_name: "A"
duration: "1h"
```
