"""
Locust Load Test for Priority Queue
Improved version with efficient task status checking
"""

from locust import HttpUser, task, between, events
import time
import json
import requests
from datetime import datetime
from collections import deque

# Global variables
task_latencies = []
task_types = []
all_submitted_tasks = []
api_host = None

class TaskQueueUser(HttpUser):
    """Simulates a client submitting tasks to the Priority queue"""
    
    wait_time = between(1, 1)
    
    # Shared deque for efficient checking
    pending_tasks = deque()
    
    def on_start(self):
        global api_host
        api_host = self.host
    
    @task(5)  # 80% short jobs
    def submit_short_task(self):
        self._submit_task("short")
    
    @task(5)  # 20% long jobs
    def submit_long_task(self):
        self._submit_task("long")
    
    def _submit_task(self, job_type):
        """Submit a task to Priority queue"""
        submit_time = time.time()
        
        with self.client.post(
            "/task/pq",  # Priority queue endpoint
            json={"job_type": job_type},
            catch_response=True,
            name="/task/pq"
        ) as response:
            if response.status_code == 201:
                try:
                    data = response.json()
                    task_id = data["task"]["id"]
                    
                    task_info = {
                        "id": task_id,
                        "job_type": job_type,
                        "submit_time": submit_time
                    }
                    
                    self.pending_tasks.append(task_info)
                    all_submitted_tasks.append(task_info)
                    
                    response.success()
                except Exception as e:
                    response.failure(f"Failed to parse response: {e}")
    
    @task(3)  # Check tasks frequently
    def check_task_status(self):
        """
        Check oldest submitted tasks first (FIFO checking)
        More efficient than random selection
        """
        if not self.pending_tasks:
            return
        
        # Check oldest task first
        task_info = self.pending_tasks.popleft()
        
        with self.client.get(
            f"/task/{task_info['id']}",
            catch_response=True,
            name="/task/:id"
        ) as response:
            if response.status_code == 200:
                try:
                    data = response.json()
                    status = data.get("status")
                    
                    if status in ["success", "failed"]:
                        # Task completed!
                        latency = time.time() - task_info["submit_time"]
                        
                        task_latencies.append(latency)
                        task_types.append(task_info["job_type"])
                        
                        # Don't re-add (completed)
                    else:
                        # Still processing, check later
                        self.pending_tasks.append(task_info)
                    
                    response.success()
                except Exception as e:
                    response.failure(f"Failed to parse: {e}")
                    self.pending_tasks.append(task_info)
            else:
                # Re-add on error
                self.pending_tasks.append(task_info)


def poll_remaining_tasks(max_wait_time=600):
    """Poll remaining uncompleted tasks after test stops"""
    
    if not api_host:
        print("Error: API host not set")
        return
    
    completed_ids = set()
    for i in range(len(task_latencies)):
        # We don't store task IDs separately, so build from all_submitted_tasks
        pass
    
    # Find uncompleted tasks
    completed_count = len(task_latencies)
    uncompleted_tasks = all_submitted_tasks[completed_count:]
    
    if not uncompleted_tasks:
        print("✅ All tasks completed during test")
        return
    
    print(f"\n⏳ Polling {len(uncompleted_tasks)} uncompleted tasks...")
    print(f"Maximum wait time: {max_wait_time}s\n")
    
    start_poll_time = time.time()
    checked_count = 0
    
    while uncompleted_tasks and (time.time() - start_poll_time) < max_wait_time:
        newly_completed = []
        
        for task_info in uncompleted_tasks:
            checked_count += 1
            
            try:
                url = f"{api_host}/task/{task_info['id']}"
                response = requests.get(url, timeout=5)
                
                if response.status_code == 200:
                    data = response.json()
                    status = data.get("status")
                    
                    if status in ["success", "failed"]:
                        latency = time.time() - task_info["submit_time"]
                        
                        task_latencies.append(latency)
                        task_types.append(task_info["job_type"])
                        
                        newly_completed.append(task_info)
                        
            except Exception:
                pass
        
        # Remove completed tasks
        for task in newly_completed:
            uncompleted_tasks.remove(task)
        
        # Progress update every 100 checks or when tasks complete
        if checked_count % 100 == 0 or newly_completed:
            elapsed = time.time() - start_poll_time
            print(f"  [{elapsed:.0f}s] Completed: {len(task_latencies)}/{len(all_submitted_tasks)}, "
                  f"Remaining: {len(uncompleted_tasks)}")
        
        time.sleep(1)
    
    if uncompleted_tasks:
        print(f"\n⚠️  {len(uncompleted_tasks)} tasks did not complete")
    else:
        print(f"\n✅ All tasks completed!")


@events.test_stop.add_listener
def on_test_stop(environment, **kwargs):
    """Calculate and print summary statistics"""
    
    print("\n" + "="*60)
    print("Test stopped - checking remaining tasks...")
    print("="*60)
    
    print(f"\nDuring test:")
    print(f"  Tasks submitted: {len(all_submitted_tasks)}")
    print(f"  Tasks completed: {len(task_latencies)}")
    
    # Poll remaining tasks
    poll_remaining_tasks(max_wait_time=120)
    
    if not task_latencies:
        print("\n❌ No completed tasks for analysis")
        return
    
    print("\n" + "="*60)
    print("PRIORITY QUEUE - FINAL LATENCY ANALYSIS")
    print("="*60)
    
    sorted_latencies = sorted(task_latencies)
    n = len(sorted_latencies)
    
    p50 = sorted_latencies[int(n * 0.50)]
    p95 = sorted_latencies[int(n * 0.95)]
    p99 = sorted_latencies[int(n * 0.99)]
    avg = sum(sorted_latencies) / n
    min_lat = min(sorted_latencies)
    max_lat = max(sorted_latencies)
    
    print(f"\nTotal completed tasks: {n} / {len(all_submitted_tasks)} ({n/len(all_submitted_tasks)*100:.1f}%)")
    print(f"Minimum latency: {min_lat:.3f}s")
    print(f"Average latency: {avg:.3f}s")
    print(f"Maximum latency: {max_lat:.3f}s")
    print(f"\nPercentiles:")
    print(f"  50th percentile: {p50:.3f}s")
    print(f"  95th percentile: {p95:.3f}s")
    print(f"  99th percentile: {p99:.3f}s")
    
    # Breakdown by job type
    short_latencies = [task_latencies[i] for i, t in enumerate(task_types) if t == "short"]
    long_latencies = [task_latencies[i] for i, t in enumerate(task_types) if t == "long"]
    
    if short_latencies:
        short_sorted = sorted(short_latencies)
        print(f"\n--- Short Jobs ({len(short_latencies)} tasks) ---")
        print(f"  Median: {short_sorted[len(short_sorted)//2]:.3f}s")
        print(f"  95th percentile: {short_sorted[int(len(short_sorted)*0.95)]:.3f}s")
        print(f"  99th percentile: {short_sorted[int(len(short_sorted)*0.99)]:.3f}s")
    
    if long_latencies:
        long_sorted = sorted(long_latencies)
        print(f"\n--- Long Jobs ({len(long_latencies)} tasks) ---")
        print(f"  Median: {long_sorted[len(long_sorted)//2]:.3f}s")
        print(f"  95th percentile: {long_sorted[int(len(long_sorted)*0.95)]:.3f}s")
    
    print("\n" + "="*60)
    
    # Save results
    results = {
        "queue_type": "priority",
        "total_tasks_submitted": len(all_submitted_tasks),
        "total_tasks_completed": n,
        "completion_rate": n / len(all_submitted_tasks),
        "min_latency": min_lat,
        "average_latency": avg,
        "max_latency": max_lat,
        "p50": p50,
        "p95": p95,
        "p99": p99,
        "short_tasks": len(short_latencies),
        "long_tasks": len(long_latencies),
        "latencies": sorted_latencies,
        "timestamp": datetime.now().isoformat()
    }
    
    with open("experiment_results.json", "w") as f:
        json.dump(results, f, indent=2)
    
    print(f"\n✅ Results saved to experiment_results.json")
    print("="*60 + "\n")