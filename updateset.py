import json


# save data to json file
def store(data):
    with open('.update', 'w') as fw:
        # convert dict to string
        # json_str = json.dumps(data)
        # fw.write(json_str)
        # ||
        json.dump(data, fw)


# load json data from file
def load():
    with open('.update', 'r') as f:
        data = json.load(f)
        return data


if __name__ == "__main__":
    #json_data = '{"login":[{"username":"aa","password":"001"},{"username":"bb","password":"002"}],"register":[{"username":"cc","password":"003"},{"username":"dd","password":"004"}]}'
    # json data struct to dict
    # data = json.loads(json_data)
    # store(data)

    data = load()
    for k in data:
        print(k)
