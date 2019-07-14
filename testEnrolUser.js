async function main(username, password){
  try {
    let ret = await enrol(username, password);
    console.log("returned ", ret);
  } catch (error) {
    console.log(error);
  }
}

const enrol = require('./enrolUser').enrolUser;

const args = process.argv;
console.log(args);
let username = "";
if(args.length == 3) {
  username = args[2];
} else {
  console.log("Usage: node enrolUser.js <username>");
  return;
}

const readline = require('readline-sync');
let password = readline.question("Enter password for user " + username + ": ", {
  hideEchoBack: true
});

main()
  .catch(err => console.log(err))
