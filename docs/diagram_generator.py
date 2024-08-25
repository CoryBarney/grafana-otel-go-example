from diagrams import Diagram, Edge
from diagrams.programming.language import Go
from diagrams.custom import Custom
from diagrams.onprem.monitoring import Grafana, Prometheus
from diagrams.onprem.tracing import Jaeger

with Diagram("Go Application with Observability Stack", show=False, direction="TB"):
    locust = Custom("Locust", "./locust_logo.png")
    app = Go("Go Application")
    otel = Custom("OpenTelemetry", "./opentelemetry_logo.png")
    prometheus = Prometheus("Prometheus")
    jaeger = Jaeger("Jaeger")
    grafana = Grafana("Grafana")

    locust >> Edge(label="HTTP") >> app
    app >> Edge(label="metrics") >> prometheus
    app >> Edge(label="traces") >> otel
    otel >> jaeger
    prometheus >> grafana
    jaeger >> grafana