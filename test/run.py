import signal
import time
from datetime import datetime

start_time = time.time()


def signal_handler(sig, frame):
    end_time = time.time()
    elapsed_time = end_time - start_time
    end_datetime = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
    print(f"\n程序运行时间: {elapsed_time:.2f}秒")
    print(f"结束时间：{end_datetime}")
    print(f"结束信号：{sig}")
    exit(0)


for sig in [signal.SIGINT, signal.SIGTERM, signal.SIGHUP]:
    signal.signal(sig, signal_handler)

if __name__ == "__main__":
    print("程序开始运行...")
    print(f"开始时间：{datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")

    while True:
        time.sleep(1)
        print(f"{datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")
