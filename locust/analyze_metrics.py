#!/usr/bin/env python3
"""
Analyze and visualize Experiment 1 metrics
Combines latency data with CPU/memory utilization
"""

import json
import matplotlib.pyplot as plt
import numpy as np
from datetime import datetime

def load_latency_data(filename):
    """Load latency results from Locust test"""
    with open(filename) as f:
        return json.load(f)

def load_metrics_data(filename):
    """Load CPU/memory metrics collected during test"""
    metrics = []
    with open(filename) as f:
        for line in f:
            if line.strip():
                metrics.append(json.loads(line))
    return metrics

def plot_comparison(fifo_data, priority_data):
    """Create comparison plots for FIFO vs Priority"""
    
    fig, axes = plt.subplots(2, 2, figsize=(14, 10))
    fig.suptitle('Experiment 1: FIFO vs Priority Queue Comparison', fontsize=16, fontweight='bold')
    
    # 1. Latency Distribution (Box Plot)
    ax1 = axes[0, 0]
    fifo_latencies = fifo_data['latencies']
    priority_latencies = priority_data['latencies']
    
    ax1.boxplot([fifo_latencies, priority_latencies], labels=['FIFO', 'Priority'])
    ax1.set_ylabel('Latency (seconds)')
    ax1.set_title('Latency Distribution')
    ax1.grid(True, alpha=0.3)
    
    # 2. Percentile Comparison (Bar Chart)
    ax2 = axes[0, 1]
    metrics = ['50th', '95th', '99th']
    fifo_percentiles = [fifo_data['p50'], fifo_data['p95'], fifo_data['p99']]
    priority_percentiles = [priority_data['p50'], priority_data['p95'], priority_data['p99']]
    
    x = np.arange(len(metrics))
    width = 0.35
    
    ax2.bar(x - width/2, fifo_percentiles, width, label='FIFO', alpha=0.8)
    ax2.bar(x + width/2, priority_percentiles, width, label='Priority', alpha=0.8)
    ax2.set_ylabel('Latency (seconds)')
    ax2.set_title('Percentile Latencies')
    ax2.set_xticks(x)
    ax2.set_xticklabels(metrics)
    ax2.legend()
    ax2.grid(True, alpha=0.3, axis='y')
    
    # 3. CDF (Cumulative Distribution Function)
    ax3 = axes[1, 0]
    
    fifo_sorted = np.sort(fifo_latencies)
    priority_sorted = np.sort(priority_latencies)
    
    fifo_cdf = np.arange(1, len(fifo_sorted) + 1) / len(fifo_sorted)
    priority_cdf = np.arange(1, len(priority_sorted) + 1) / len(priority_sorted)
    
    ax3.plot(fifo_sorted, fifo_cdf, label='FIFO', linewidth=2)
    ax3.plot(priority_sorted, priority_cdf, label='Priority', linewidth=2)
    ax3.set_xlabel('Latency (seconds)')
    ax3.set_ylabel('Cumulative Probability')
    ax3.set_title('Latency CDF (Shows Tail Behavior)')
    ax3.legend()
    ax3.grid(True, alpha=0.3)
    ax3.set_xlim(0, min(max(fifo_sorted), 10))  # Limit x-axis for readability
    
    # 4. Summary Stats Table
    ax4 = axes[1, 1]
    ax4.axis('off')
    
    table_data = [
        ['Metric', 'FIFO', 'Priority', 'Improvement'],
        ['50th %ile', f"{fifo_data['p50']:.3f}s", f"{priority_data['p50']:.3f}s", 
         f"{((fifo_data['p50']-priority_data['p50'])/fifo_data['p50']*100):.1f}%"],
        ['95th %ile', f"{fifo_data['p95']:.3f}s", f"{priority_data['p95']:.3f}s",
         f"{((fifo_data['p95']-priority_data['p95'])/fifo_data['p95']*100):.1f}%"],
        ['99th %ile', f"{fifo_data['p99']:.3f}s", f"{priority_data['p99']:.3f}s",
         f"{((fifo_data['p99']-priority_data['p99'])/fifo_data['p99']*100):.1f}%"],
        ['Average', f"{fifo_data['average_latency']:.3f}s", f"{priority_data['average_latency']:.3f}s",
         f"{((fifo_data['average_latency']-priority_data['average_latency'])/fifo_data['average_latency']*100):.1f}%"],
        ['Submitted Tasks', str(fifo_data['total_tasks_submitted']), str(priority_data['total_tasks_submitted']), '-'],
        ['Completed Tasks', str(fifo_data['total_tasks_completed']), str(priority_data['total_tasks_completed']), '-']
    ]
    
    table = ax4.table(cellText=table_data, cellLoc='center', loc='center',
                      colWidths=[0.25, 0.25, 0.25, 0.25])
    table.auto_set_font_size(False)
    table.set_fontsize(10)
    table.scale(1, 2)
    
    # Style header row
    for i in range(4):
        table[(0, i)].set_facecolor('#4CAF50')
        table[(0, i)].set_text_props(weight='bold', color='white')
    
    ax4.set_title('Summary Statistics', fontweight='bold', pad=20)
    
    plt.tight_layout()
    plt.savefig('experiment1_comparison.png', dpi=300, bbox_inches='tight')
    print("\n✅ Comparison plot saved to experiment1_comparison.png")
    plt.close()


def plot_resource_utilization(fifo_metrics, priority_metrics):
    """Plot CPU and memory utilization over time"""
    
    fig, axes = plt.subplots(2, 2, figsize=(14, 10))
    fig.suptitle('Resource Utilization: FIFO vs Priority', fontsize=16, fontweight='bold')
    
    # Extract data
    def extract_metric(metrics, key):
        values = []
        for m in metrics:
            try:
                val = float(m.get(key, 0))
                if val > 0:  # Filter out "None" values
                    values.append(val)
            except (ValueError, TypeError):
                pass
        return values
    
    fifo_worker_cpu = extract_metric(fifo_metrics, 'worker_cpu')
    priority_worker_cpu = extract_metric(priority_metrics, 'worker_cpu')
    
    fifo_redis_cpu = extract_metric(fifo_metrics, 'redis_cpu')
    priority_redis_cpu = extract_metric(priority_metrics, 'redis_cpu')
    
    # Plot Worker CPU
    # ax1 = axes[0, 0]
    # if fifo_worker_cpu and priority_worker_cpu:
    #     ax1.plot(range(len(fifo_worker_cpu)), fifo_worker_cpu, label='FIFO', linewidth=2, marker='o')
    #     ax1.plot(range(len(priority_worker_cpu)), priority_worker_cpu, label='Priority', linewidth=2, marker='s')
    #     ax1.set_ylabel('CPU Utilization (%)')
    #     ax1.set_xlabel('Time (30s intervals)')
    #     ax1.set_title('Worker CPU Utilization')
    #     ax1.legend()
    #     ax1.grid(True, alpha=0.3)
    #     ax1.set_ylim(0, 100)
    
    # Plot Redis CPU
    ax2 = axes[0, 0]
    if fifo_redis_cpu and priority_redis_cpu:
        ax2.plot(range(len(fifo_redis_cpu)), fifo_redis_cpu, label='FIFO', linewidth=2, marker='o')
        ax2.plot(range(len(priority_redis_cpu)), priority_redis_cpu, label='Priority', linewidth=2, marker='s')
        ax2.set_ylabel('CPU Utilization (%)')
        ax2.set_xlabel('Time (30s intervals)')
        ax2.set_title('Redis CPU Utilization')
        ax2.legend()
        ax2.grid(True, alpha=0.3)
        ax2.set_ylim(0, 100)
    
    # Average CPU comparison
    ax3 = axes[1, 0]
    categories = ['Redis CPU']
    fifo_avgs = [np.mean(fifo_redis_cpu) if fifo_redis_cpu else 0]
    priority_avgs = [np.mean(priority_redis_cpu) if priority_redis_cpu else 0]
    
    x = np.arange(len(categories))
    width = 0.35
    
    ax3.bar(x - width/2, fifo_avgs, width, label='FIFO', alpha=0.8)
    ax3.bar(x + width/2, priority_avgs, width, label='Priority', alpha=0.8)
    ax3.set_ylabel('Average CPU (%)')
    ax3.set_title('Average CPU Utilization')
    ax3.set_xticks(x)
    ax3.set_xticklabels(categories)
    ax3.legend()
    ax3.grid(True, alpha=0.3, axis='y')
    
    # Summary table
    ax4 = axes[1, 1]
    ax4.axis('off')
    
    table_data = [
        ['Metric', 'FIFO', 'Priority'],
        ['Avg Redis CPU', f"{np.mean(fifo_redis_cpu):.1f}%" if fifo_redis_cpu else "N/A",
         f"{np.mean(priority_redis_cpu):.1f}%" if priority_redis_cpu else "N/A"]
    ]
    
    table = ax4.table(cellText=table_data, cellLoc='center', loc='center',
                      colWidths=[0.4, 0.3, 0.3])
    table.auto_set_font_size(False)
    table.set_fontsize(11)
    table.scale(1, 2)
    
    for i in range(3):
        table[(0, i)].set_facecolor('#2196F3')
        table[(0, i)].set_text_props(weight='bold', color='white')
    
    ax4.set_title('Resource Utilization Summary', fontweight='bold', pad=20)
    
    plt.tight_layout()
    plt.savefig('experiment1_resource_utilization.png', dpi=300, bbox_inches='tight')
    print("✅ Resource utilization plot saved to experiment1_resource_utilization.png")
    plt.close()


if __name__ == "__main__":
    # Load data
    try:
        fifo_latency = load_latency_data('fifo_experiment_results.json')
        priority_latency = load_latency_data('priority_experiment_results.json')
        
        print("Generating latency comparison plots...")
        plot_comparison(fifo_latency, priority_latency)
        
        # Load metrics if available
        try:
            fifo_metrics = load_metrics_data('fifo_metrics.json')
            priority_metrics = load_metrics_data('priority_metrics.json')
            
            print("Generating resource utilization plots...")
            plot_resource_utilization(fifo_metrics, priority_metrics)
        except FileNotFoundError:
            print("⚠️  Metrics files not found, skipping resource utilization plots")
        
        print("\n✅ All plots generated successfully!")
        print("   - experiment1_comparison.png")
        print("   - experiment1_resource_utilization.png")
        
    except FileNotFoundError as e:
        print(f"❌ Error: {e}")
        print("Make sure you've run the experiments first!")