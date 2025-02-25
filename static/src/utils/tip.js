import {ElMessage} from "element-plus";

const prefix = "";

function _sendErrorMessage(message) {
    ElMessage({message: prefix + message, type: 'error'});
}

function _sendSuccessMessage(message) {
    ElMessage({message: prefix + message, type: 'success'});
}

// limit the frequency of function calls
export function throttle(fn, interval) {
    let last = 0;   // 维护上次执行的时间
    return function (...args) {
        const now = Date.now();
        if (now - last >= interval) {
            last = now;
            fn.apply(this, args);  // 使用 apply 来传递参数数组
        }
    };
}

// 使用防抖函数包装，1s 内只能发送一次消息
export const sendErrorMessage = throttle(_sendErrorMessage, 1000);
export const sendSuccessMessage = throttle(_sendSuccessMessage, 1000);
