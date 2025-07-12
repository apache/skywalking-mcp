# MQE AI Model Understanding Guide

## Core Understanding Principles

When users describe monitoring requirements in natural language, you need to:

1. **Understand user intent**, not just mechanically match keywords
2. **Infer implicit information**, such as time ranges, aggregation methods, etc.
3. **Handle errors gracefully**, understand spelling mistakes or non-standard expressions
4. **Provide explanations**, explain the meaning of generated MQE expressions

## Natural Language Understanding Strategies

### 1. Intent Recognition Patterns

#### 1.1 Query Type Intents
- "view/show/get/display..." → Simple query
- "monitor/observe/track/watch..." → Continuous query
- "analyze/compare/contrast..." → Complex query
- "alert/alarm/notify..." → Conditional judgment

#### 1.2 Metric Recognition
Users may express the same metric in various ways:

**Response Time**:
- response time, latency, delay, time spent, RT, latency
- → `service_resp_time` or `endpoint_resp_time`

**Success Rate**:
- success rate, availability, SLA, uptime, health
- → `service_sla`

**Call Volume**:
- call volume, request count, traffic, QPS, TPS, throughput
- → `service_cpm` (needs context-based conversion)

**Error Rate**:
- error rate, failure rate, exception rate
- → `100 - (service_sla * 100)` or use status_code metrics

### 2. Time Range Understanding

#### 2.1 Relative Time
**Past Time (negative duration)**:
- "recently/past/just now" + number + unit
  - "recent 5 minutes" → `duration: "-5m"`
  - "past one hour" → `duration: "-1h"`
  - "recent one day" → `duration: "-24h"` or `duration: "-1d"`

**Future Time (positive duration)**:
- "next/future/upcoming/coming" + number + unit
  - "next 5 minutes" → `duration: "5m"`
  - "upcoming one hour" → `duration: "1h"`
  - "next day" → `duration: "24h"` or `duration: "1d"`

#### 2.2 Fuzzy Time
**Past-oriented (negative)**:
- "just now" → `duration: "-5m"` (defaults to 5 minutes)
- "recently" → `duration: "-1h"` (defaults to 1 hour)

**Future-oriented (positive)**:
- "soon" → `duration: "5m"` (defaults to next 5 minutes)
- "later" → `duration: "1h"` (defaults to next 1 hour)

#### 2.3 Absolute Time
- "from...to..." → use start and end
- "January 1, 2025, 10 am to 11 am" → convert to standard format

### 3. Inference of Aggregation Methods

#### 3.1 Default Aggregation
- Query response time → usually use `avg()`
- Query error count → usually use `sum()`
- Query peak value → use `max()`
- Query minimum value → use `min()`

#### 3.2 Explicit Aggregation
- "average response time" → `avg(service_resp_time)`
- "maximum latency" → `max(service_resp_time)`
- "total call volume" → `sum(service_cpm)`
- "median response time" → `service_percentile{p='50'}`

### 4. Condition Understanding

#### 4.1 Threshold Judgment
- "exceeds/greater than/higher than" → `>`
- "less than/lower than" → `<`
- "reaches/equal to" → `>=` or `==`
- "between...and..." → combine `>` and `<`

#### 4.2 Unit Conversion
- "exceeds 1 second" → `> 1000` (convert to milliseconds)
- "less than 90%" → `< 90` (after converting SLA: `service_sla * 100 < 90`)
- "100 times per second" → convert to CPM: `> 6000`

### 5. Complex Query Patterns

#### 5.1 Multi-Condition Combination
User: "response time exceeds 3 seconds and low call volume"
- Identify two conditions
- "low" requires inferred threshold, such as `< 100`
- Generate: `service_resp_time > 3000 && service_cpm < 100`

#### 5.2 Percentile Query
User: "check P95 and P99 response time"
- Identify percentile requirement
- Generate: `service_percentile{p='95,99'}`

#### 5.3 Sorting Requirement
User: "slowest 10 services"
- Identify sorting and quantity
- Generate: `top_n(service_resp_time, 10, des)`

### 6. Error Tolerance Handling

#### 6.1 Spelling Errors
- "sevice" → "service"
- "responce" → "response"
- "avrage" → "average"

#### 6.2 Abbreviation Understanding
- "RT" → Response Time
- "QPS" → Queries Per Second (convert to CPM)
- "P95" → 95th percentile

#### 6.3 Colloquial Expressions
- "this service is slow" → query response time
- "traffic suddenly increased" → use increase() function
- "higher than usual" → use baseline() function

### 7. Context Understanding

#### 7.1 Service Recognition
- If the user mentions a specific service name, extract and use it
- If not specified, may need to:
  - Query aggregate data for all services
  - Use top_n to find related services

#### 7.2 Level Understanding
- "database" → `layer: "DATABASE"`
- "gateway" → possibly a specific service name
- "microservice" → `layer: "GENERAL"`

### 8. Response Generation Strategy

#### 8.1 Explaining Generated Expressions
```
User: "check the recent average response time for service A"
Response: "I will query the average response time for service A in the last hour:"
Expression: avg(service_resp_time)
Parameters: service_name="A", duration="-1h"
```

#### 8.2 Providing Alternatives
```
User: "service is slow"
Response: "I understand you want to check service performance issues. Here are several query options:
1. Check average response time: avg(service_resp_time)
2. Check response time percentiles: service_percentile{p='50,75,90,95,99'}
3. Check slowest services: top_n(service_resp_time, 10, des)"
```

#### 8.3 Suggesting Optimizations
```
User: "requests per second"
Response: "service_cpm returns calls per minute, I'll help you convert to per second:"
Expression: service_cpm / 60
```

### 9. Common Scenario Templates

#### 9.1 Performance Analysis
- "service performance analysis" → combine multiple metrics
  ```
  Response time: avg(service_resp_time)
  Call volume: service_cpm
  Success rate: service_sla * 100
  ```

#### 9.2 Alert Rules
- "response time alert" → generate comparison expression
  ```
  avg(service_resp_time) > 1000
  ```

#### 9.3 Trend Analysis
- "traffic trend" → use trend functions
  ```
  rate(service_cpm, 5)  # rate of change over 5 minutes
  ```

### 10. Error Handling Tips

When unable to understand user intent:
1. Ask for specifics: "Do you want to check response time or call volume?"
2. Provide examples: "For example: check average response time for service A"
3. List available options: "Available metrics include: response time, call volume, success rate, etc."

## Practical Examples

### Example 1: Fuzzy Query
User: "service seems to have issues"
Analysis:
- Intent: diagnose problems
- Suggest querying multiple key metrics

Response:
```
Let me check the key service metrics:
1. Success rate: service_sla * 100
2. Response time: avg(service_resp_time)
3. Call volume: service_cpm
These metrics can help identify the issue.
```

### Example 2: Complex Conditions
User: "find services with slow response but high traffic"
Analysis:
- Needs to satisfy two conditions
- "slow" and "high" need threshold definitions

Response:
```
expression: "service_resp_time > 1000 && service_cpm > 1000"
Explanation: Find services with response time over 1 second and over 1000 calls per minute
You can also use top_n to find such services.
```

### Example 3: Time Comparison
User: "compare today's traffic with yesterday"
Analysis:
- Needs two time period queries
- May need to calculate difference or ratio

Response:
```
Need to execute two queries:
1. Today: start="today 00:00", end="now", expression="avg(service_cpm)"
2. Yesterday: start="yesterday 00:00", end="yesterday 24:00", expression="avg(service_cpm)"
Then compare the results.
```

## Best Practices

1. **Proactive Inference**: Users may not know specific metric names, infer from descriptions
2. **Provide Context**: Explain why choosing a certain expression
3. **Flexible Handling**: Same requirement may have multiple implementations
4. **Educate Users**: Include MQE knowledge in responses to help users learn
5. **Safe Defaults**: When uncertain, choose more conservative query approaches
