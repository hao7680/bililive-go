/*
 * @Author: Jmeow
 * @Date: 2020-01-28 11:25:44
 * @Description: common utils
 */

async function customFetch(url: string, options?: RequestInit): Promise<any> {
    const response = await fetch(url, options);
    const data = await response.json();

    if (response.ok) {
        return data;
    } else {
        if (data && data.err_msg) {
            Utils.alertError(data.err_msg);
        } else {
            throw new Error('Unknown error');
        }
    }
}


class Utils {
    /**
     * Get request
     * @param url URL
     */
    requestGet(url: string) {
        return customFetch(url);
    }

    /**
     * Post request
     * @param url URL
     * @param body Request body
     */
    requestPost(url: string, body?: object) {
        return customFetch(url, {
            method: 'POST',
            body: JSON.stringify(body),
            headers: new Headers({
                'Content-Type': 'application/json'
            })
        });
    }

    /**
     * Post request
     * @param url URL
     * @param body Request body
     */
    requestPut(url: string, body?: object) {
        return customFetch(url, {
            method: 'PUT',
            body: JSON.stringify(body),
            headers: new Headers({
                'Content-Type': 'application/json'
            })
        })
    }

    /**
     * Delete request
     * @param url URL
     */
    requestDelete(url: string) {
        return customFetch(url, {
            method: 'DELETE'
        });
    }

    /**
     * Show Error 
     * @param err error Object
     */
    static alertError(err?: any) {
        alert(err ? err : "Server Error!");
    }

    static byteSizeToHumanReadableFileSize(size: number): string {
        if (!size) {
            return "0";
        }
        const i = Math.floor(Math.log(size) / Math.log(1024));
        const ret = Number((size / Math.pow(1024, i)).toFixed(2)) + " " + ['B', 'kB', 'MB', 'GB', 'TB'][i];
        return ret;
    }

    static timestampToHumanReadable(timestamp: number): string {
        const date = new Date(timestamp * 1000);
        const year = date.getFullYear().toString().padStart(4, "0");
        const month = (date.getMonth() + 1).toString().padStart(2, "0");
        const day = date.getDate().toString().padStart(2, "0");
        const hour = date.getHours().toString().padStart(2, "0");
        const min = date.getMinutes().toString().padStart(2, "0");
        const sec = date.getSeconds().toString().padStart(2, "0");
        return `${year}-${month}-${day} ${hour}:${min}:${sec}`;
    }
}

export default Utils;
