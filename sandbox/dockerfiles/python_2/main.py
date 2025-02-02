import requests

def fetch_example():
    response = requests.get("https://jsonplaceholder.typicode.com/todos/1")
    if response.status_code == 200:
        print("Данные:", response.json())
    else:
        print("Ошибка:", response.status_code)

if __name__ == "__main__":
    fetch_example()
