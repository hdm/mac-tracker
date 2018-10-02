#!/usr/bin/env ruby

require 'find'
require 'fileutils'
require 'json'

def usage
	$stderr.puts "usage: #{$0} ./deepmac-journal-snapshot/ <consolidated.json>"
	exit(0)
end

@oui = {}

dir = ARGV.shift || usage()
out = ARGV.shift || usage()

Find.find(dir).each do |f|
	next unless File.file?(f)
	info = JSON.parse(File.read(f))

	next unless info
	next unless info['recs']
	data = []

	addr = nil
	mask = nil 
	info['recs'].each do |r|
		clean = {}
		addr ||= r["OUI"].downcase.ljust(12, "0")
		mask ||= r["OUISize"].to_i
		clean["d"] = r["EventDate"]
		clean["t"] = r["EventType"]
		clean["a"] = r["OrgAddress"].to_s.gsub("\\n", "\n")
		clean["c"] = r["OrgCountry"]
		clean["o"] = r["OrgName"]
		data << clean
	end

	okey = addr + "/" + mask.to_s

	@oui[okey] ||= {}
	@oui[okey] = data
end

File.open(out, "w") do |fd|
	fd.write(JSON.dump(@oui))
end