# logbeat

logbeat is an "EKG" for a website. Consumes log entries from stdin that have a php_time=(float) component, and outputs audio with some basic health stats

- Freq in HZ = 90th percentile response time in ms
- Beats / minute = requests / second
