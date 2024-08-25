from locust import HttpUser, task, between
import json
import random

class GoAppUser(HttpUser):
    wait_time = between(1, 3)  # Wait 1-3 seconds between tasks

    @task(2)
    def post_sentence(self):
        payload = {
            "text": f"Test sentence {random.randint(1, 1000)}"
        }
        headers = {'Content-Type': 'application/json'}
        with self.client.post("/api/v1/sentence", data=json.dumps(payload), headers=headers, catch_response=True) as response:
            if response.status_code == 200:
                response.success()
            else:
                response.failure(f"Got unexpected response code: {response.status_code}")

    @task(1)
    def get_random_delay(self):
        with self.client.get("/api/v1/random-delay", catch_response=True) as response:
            if response.status_code == 200:
                response.success()
            else:
                response.failure(f"Got unexpected response code: {response.status_code}")

    def on_start(self):
        # Log the start of a new user
        print("A new user has started")