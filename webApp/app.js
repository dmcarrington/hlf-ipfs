// Run with: DEBUG=webApp:* npm start

const createError = require('http-errors');
const express = require('express');
const path = require('path');
const cookieParser = require('cookie-parser');
const logger = require('morgan');
const passport     = require('passport'),
    exphbs = require('express-handlebars'),
    bodyParser   = require('body-parser'),
    methodOverride = require('method-override'),
    session = require('express-session'),
    LdapStrategy = require('passport-ldapauth'),
    parseJson = require('parse-json');

// If this is running in a Docker container, the url
// must be the name of the container, else localhost.
var OPTS = {
  passReqToCallback: true,
  server: {
    url: 'ldap://ldap-server:389',
    bindDN: 'cn=admin,dc=example,dc=com',
    bindCredentials: 'adminpw',
    searchBase: 'dc=example,dc=com',
    searchFilter: '(cn={{username}})'
  }
};
const fileUpload = require('express-fileupload');
var app = express();

app.use(fileUpload());

passport.use(new LdapStrategy(OPTS));

// view engine setup
app.set('views', path.join(__dirname, 'views'));
app.set('view engine', 'hbs');

app.use(logger('dev'));
app.use(express.json());
app.use(express.urlencoded({ extended: false }));
app.use(cookieParser());
app.use(express.static(path.join(__dirname, 'public')));

app.use(bodyParser.json());
app.use(bodyParser.urlencoded({extended: false}));
app.use(methodOverride('X-HTTP-Method-Override'));
app.use(session({secret: 'supernova', saveUninitialized: true, resave: true}));
app.use(passport.initialize());
app.use(passport.session());

// Session-persisted message middleware
app.use(function(req, res, next){
  var err = req.session.error,
      msg = req.session.notice,
      success = req.session.success;

  delete req.session.error;
  delete req.session.success;
  delete req.session.notice;

  if (err) res.locals.error = err;
  if (msg) res.locals.notice = msg;
  if (success) res.locals.success = success;

  next();
});

// Configure express to use handlebars templates
var hbs = exphbs.create({
  defaultLayout: 'main', //we will be creating this layout shortly
});
app.engine('handlebars', hbs.engine);
app.set('view engine', 'handlebars');

//===============ROUTES=================
//displays our homepage
app.get('/', async function(req, res){
  
  let username = undefined;
  let userEnrolled = false;
  let userOriginatedTransfers = undefined;
  let userRecipientTransfers = undefined;
  if(req.session.user) {
    username = req.session.user.cn;
    
    var fabricClient = require('./config/FabricClient');
    await fabricClient.initCredentialStores();
    await fabricClient.getCertificateAuthority();
    let user = await fabricClient.getUserContext(username, true);
    if(user) {
      userEnrolled = true;

      // query the chaincode for transfers created by this user
      let fcn = "queryTransfersByOriginator";
      const args = [username];
      const queryChaincode = require('./invoke.js').queryChaincode;
      let chaincodeContent = await queryChaincode(fabricClient, fcn, args);
      // parse the results and present as a table with a link to each IPFS entry
      let response = chaincodeContent.payload.responses[0];
      let start = response.indexOf("[");
      let responseJsonSource = response.substring(start, response.length);
      responseJsonSource = responseJsonSource.replace(/\u0000/gu, "");
      try{
        const responseJson = parseJson(responseJsonSource);
        userOriginatedTransfers = responseJson;
      } catch (error) {
        console.log(error);
      }

      // Repeat the query for transfers assigned to us
      fcn = "queryTransfersByRecipient";
      chaincodeContent = await queryChaincode(fabricClient, fcn, args);
      
      // parse the results and present as a table with a link to each IPFS entry
      response = chaincodeContent.payload.responses[0];
      start = response.indexOf("[");
      responseJsonSource = response.substring(start, response.length);
      responseJsonSource = responseJsonSource.replace(/\u0000/gu, "");
      try{
        const responseJson = parseJson(responseJsonSource);
        userRecipientTransfers = responseJson;
      } catch (error) {
        console.log(error);
      }
    }
  }
  res.render('home', {
    user: username, 
    userEnrolled: userEnrolled, 
    userOriginatedTransfers: userOriginatedTransfers, 
    userRecipientTransfers: userRecipientTransfers
  });

});

//displays our signup page
app.get('/signin', function(req, res){
  res.render('signin');
});

// Following the successful download of a file, mark the transfer as being complete in the chaincode
app.post('/transferComplete', async function(req, res) {
  console.log('in transferComplete');
  console.log("user: ", req.session.user);
  console.log(req.body);

  const username = req.session.user.cn;
  const uuid = req.body.uuid;
  // Invoke the markTransferAsRead function as this user with this uuid
  const fcn = "markTransferAsRead";
  const args = [uuid];
  
  var fabricClient = require('./config/FabricClient');
  await fabricClient.initCredentialStores();
  await fabricClient.getCertificateAuthority();
  await fabricClient.getUserContext(username.trim(), true);
  const proposeTransaction = require('./invoke.js').proposeTransaction;
  const proposalObject = await proposeTransaction(fabricClient, fcn, args);
  if (!proposalObject.success){
    res.send({
      success: 500, 
      message: "Unable to propose your transaction"});
  }
      
  const commitTransaction = require('./invoke.js').commitTransaction;
  const committedObject = await commitTransaction(fabricClient, 
    proposalObject.payload.txId, 
    proposalObject.payload.proposalResponses, 
    proposalObject.payload.proposal);
  if (!committedObject.success){
    res.send({
    success: 500, 
    message: "Unable to commit your transaction"});
  }

  if (committedObject.payload.commitStatus == 'SUCCESS'){
    console.log("success!");
  }
  setTimeout(function(){
    res.redirect("/");
  },5000);
});

// Commit the selected file to IPFS, and record the transaction in our chaincode
app.post('/upload-file', async function(req, res) {

    // Got our file with its content courtesy of express-fileupload
    // file.data contains the actual contents of the file
    
    // TODO: we shouldn't really be reading the entire content of the file asynchronously as part
    // of this endpoint, but it will do for now as long as we only test with small files :)
    const file = req.files.uploadFilename;
    const fileContent = file.data;

    const writeToIPFS = require('./ipfs').writeToIPFS;
    const commitHash = await writeToIPFS(fileContent);
    if(commitHash === 0) {
      res.render('home', {user: req.session.user.cn, message: "Failed to commit file to IPFS"});
    }
    else {
      // File was successfully committed to IPFS, now log the transfer in our chaincode
      const recipient = req.body.recipient;
      const msg = "File commited sucessfully: " + commitHash;
      const fcn = "createTransfer";
      const args = [req.session.user.cn.trim(), commitHash, recipient, file.name];
  
      var fabricClient = require('./config/FabricClient');
      await fabricClient.initCredentialStores();
      await fabricClient.getCertificateAuthority();
      await fabricClient.getUserContext(req.session.user.cn.trim(), true);
      const proposeTransaction = require('./invoke.js').proposeTransaction;
      const proposalObject = await proposeTransaction(fabricClient, fcn, args);
      if (!proposalObject.success){
        res.send({
          success: 500, 
          message: "Unable to propose your transaction"});
      }
      
      const commitTransaction = require('./invoke.js').commitTransaction;
      const committedObject = await commitTransaction(fabricClient, 
        proposalObject.payload.txId, 
        proposalObject.payload.proposalResponses, 
        proposalObject.payload.proposal);
      if (!committedObject.success){
        res.send({
          success: 500, 
          message: "Unable to commit your transaction"});
      }

      if (committedObject.payload.commitStatus == 'SUCCESS'){
        console.log("success!");
      }
      setTimeout(function(){
        res.redirect("/");
      },5000);
      
    }

});

// Enrol LDAP user into Fabric
app.post('/local-reg', async function(req, res) {
    const username = req.body.username;
    const password = req.body.password;
    const enrolUser = require('./enrolUser').enrolUser;
    const result = await enrolUser(username, password);
   
    if(result === 'ok') {
      res.render('signin', {message: "User enrolled successfully"})
    }
    else {
      res.render('signin', {message: "Failed to enrol user with Hyperledger Fabric network. The \
      username and password may not be valid on the LDAP server, or the user may already be installed."})
    }
  }
);

//logs user out of site, deleting them from the session, and returns to homepage
app.get('/logout', function(req, res){
  var name = req.session.user.cn;
  console.log("LOGGING OUT " + req.session.user.cn)
  req.logout();
  req.session.user = null;
  res.redirect('/');
  req.session.notice = "You have successfully been logged out " + name + "!";
});

// Passport-ldapauth of a user to access the web app
app.post('/login', async function(req, res, next) {
  passport.authenticate('ldapauth', async function (err, user, info){
    if(user){
      console.log("successfully authenticated user:", user);
      req.session.secret = req.body.password;
      req.session.user = user;
      res.redirect('/');
    } else {
      res.redirect('/signin');
    }
  })(req, res, next);
});

// catch 404 and forward to error handler
app.use(function(req, res, next) {
  next(createError(404));
});

// error handler
app.use(function(err, req, res, next) {
  // set locals, only providing error in development
  res.locals.message = err.message;
  res.locals.error = req.app.get('env') === 'development' ? err : {};

  // render the error page
  res.status(err.status || 500);
  res.render('error');
});

module.exports = app;
